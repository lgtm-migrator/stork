package dbmodel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
	storkutil "isc.org/stork/util"
)

// Custom statistic type to redefine JSON marshalling.
type SubnetStats map[string]interface{}

// Subnet statistics may contain the integer number from the int64
// (or uint64) range (max value is 2^63-1 (2^64-1)). The value returned by
// the Kea and stored in the Postgres database is exact. But when the
// frontend fetches this data, it deserializes it using the standard JSON.parse
// function. This function treats all number literals as floating double-precision
// numbers. This type can exact handle integers up to (2^53 - 1); greater numbers
// are inaccurate.
// All the numeric statistics are serialized to string and next deserialized using
// a custom function to avoid losing the precision.
//
// It doesn't use the pointer to receiver type for compatibility with gopg serialization
// during inserting to the database.
func (s SubnetStats) MarshalJSON() ([]byte, error) {
	if s == nil {
		return json.Marshal(nil)
	}

	toMarshal := make(map[string]interface{}, len(s))

	for k, v := range s {
		switch value := v.(type) {
		case *big.Int:
			toMarshal[k] = value.String()
		case int64, uint64:
			toMarshal[k] = fmt.Sprint(value)
		default:
			toMarshal[k] = value
		}
	}

	return json.Marshal(toMarshal)
}

// An interface for a wrapper of subnet statistics that encapsulates the
// utilization calculations. It corresponds to the
// `statisticscounter.subnetStats` interface and prevents the dependency cycle.
type utilizationStats interface {
	GetAddressUtilization() float64
	GetDelegatedPrefixUtilization() float64
	GetStatistics() SubnetStats
}

// Deserialize statistics and convert back the strings to int64 or uint64.
// I assume that the statistics will always contain numeric data, no string
// that look like integers.
// During the serialization we lost the original data type of the number.
// We assume that strings with positive numbers are uint64 and negative numbers are int64.
func (s *SubnetStats) UnmarshalJSON(data []byte) error {
	toUnmarshal := make(map[string]interface{})
	err := json.Unmarshal(data, &toUnmarshal)
	if err != nil {
		return err
	}

	if toUnmarshal == nil {
		*s = nil
		return nil
	}

	if *s == nil {
		*s = SubnetStats{}
	}

	for k, v := range toUnmarshal {
		vStr, ok := v.(string)
		if !ok {
			(*s)[k] = v
			continue
		}

		vUint64, err := strconv.ParseUint(vStr, 10, 64)
		if err == nil {
			(*s)[k] = vUint64
			continue
		}

		vInt64, err := strconv.ParseInt(vStr, 10, 64)
		if err == nil {
			(*s)[k] = vInt64
			continue
		}

		vBigInt, ok := new(big.Int).SetString(vStr, 10)
		if ok {
			(*s)[k] = vBigInt
			continue
		}

		(*s)[k] = v
	}

	return nil
}

// This structure holds subnet information retrieved from an app. Multiple
// DHCP server apps may be configured to serve leases in the same subnet.
// For the same subnet configured on different DHCP server there will be
// a separate instance of the LocalSubnet structure. Apart from possibly
// different local subnet id between different apos there will also be
// other information stored here, e.g. statistics for the particular
// subnet retrieved from the given app. Multiple local subnets can be
// associated with a single global subnet depending on how many daemons
// serve the same subnet.
type LocalSubnet struct {
	SubnetID      int64   `pg:",pk"`
	DaemonID      int64   `pg:",pk"`
	Daemon        *Daemon `pg:"rel:has-one"`
	Subnet        *Subnet `pg:"rel:has-one"`
	LocalSubnetID int64

	Stats            SubnetStats
	StatsCollectedAt time.Time
}

// Reflects IPv4 or IPv6 subnet from the database.
type Subnet struct {
	ID          int64
	CreatedAt   time.Time
	Prefix      string
	ClientClass string

	SharedNetworkID int64
	SharedNetwork   *SharedNetwork `pg:"rel:has-one"`

	AddressPools []AddressPool `pg:"rel:has-many"`
	PrefixPools  []PrefixPool  `pg:"rel:has-many"`

	LocalSubnets []*LocalSubnet `pg:"rel:has-many"`

	Hosts []Host `pg:"rel:has-many"`

	AddrUtilization  int16
	PdUtilization    int16
	Stats            SubnetStats
	StatsCollectedAt time.Time
}

// Hook executed after inserting a subnet to the database. It updates subnet
// id on the hosts belonging to this subnet.
func (s *Subnet) AfterInsert(ctx context.Context) error {
	if s != nil && s.ID != 0 {
		for i := range s.Hosts {
			s.Hosts[i].SubnetID = s.ID
		}
	}
	return nil
}

// Return family of the subnet.
func (s *Subnet) GetFamily() int {
	family := 4
	if strings.Contains(s.Prefix, ":") {
		family = 6
	}
	return family
}

// Add address and prefix pools from the subnet instance into the database
// in a transaction. The subnet is expected to exist in the database.
func addSubnetPools(tx *pg.Tx, subnet *Subnet) (err error) {
	if len(subnet.AddressPools) == 0 && len(subnet.PrefixPools) == 0 {
		return nil
	}
	// Add address pools first.
	for i, p := range subnet.AddressPools {
		pool := p
		pool.SubnetID = subnet.ID
		_, err = tx.Model(&pool).OnConflict("DO NOTHING").Insert()
		if err != nil {
			err = pkgerrors.Wrapf(err, "problem adding address pool %s-%s for subnet with ID %d",
				pool.LowerBound, pool.UpperBound, subnet.ID)
			return err
		}
		subnet.AddressPools[i] = pool
	}
	// Add prefix pools. This should be empty for IPv4 case.
	for i, p := range subnet.PrefixPools {
		pool := p
		pool.SubnetID = subnet.ID
		_, err = tx.Model(&pool).OnConflict("DO NOTHING").Insert()
		if err != nil {
			err = pkgerrors.Wrapf(err, "problem adding prefix pool %s for subnet with ID %d",
				pool.Prefix, subnet.ID)
			return err
		}
		subnet.PrefixPools[i] = pool
	}

	return nil
}

// Adds a new subnet and its pools to the database within a transaction.
func addSubnetWithPools(tx *pg.Tx, subnet *Subnet) error {
	// Add the subnet first.
	_, err := tx.Model(subnet).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem adding new subnet with prefix %s", subnet.Prefix)
		return err
	}
	// Add the pools.
	err = addSubnetPools(tx, subnet)
	if err != nil {
		return err
	}
	return err
}

// Adds a subnet with its pools into the database. If the subnet has any
// associations with a shared network, those associations are also created
// in the database. It begins a new transaction when dbi has a *pg.DB type
// or uses an existing transaction when dbi has a *pg.Tx type.
func AddSubnet(dbi dbops.DBI, subnet *Subnet) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addSubnetWithPools(tx, subnet)
		})
	}
	return addSubnetWithPools(dbi.(*pg.Tx), subnet)
}

// Fetches the subnet and its pools by id from the database.
func GetSubnet(dbi dbops.DBI, subnetID int64) (*Subnet, error) {
	subnet := &Subnet{}
	err := dbi.Model(subnet).
		Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("SharedNetwork").
		Relation("LocalSubnets.Daemon.App.AccessPoints").
		Where("subnet.id = ?", subnetID).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting the subnet with ID %d", subnetID)
		return nil, err
	}
	return subnet, err
}

// Fetches all subnets associated with the given daemon by ID.
func GetSubnetsByDaemonID(dbi dbops.DBI, daemonID int64) ([]Subnet, error) {
	subnets := []Subnet{}

	q := dbi.Model(&subnets).
		Join("INNER JOIN local_subnet AS ls ON ls.subnet_id = subnet.id").
		Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("SharedNetwork").
		Relation("LocalSubnets.Daemon.App.AccessPoints").
		Where("ls.daemon_id = ?", daemonID)

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting subnets by daemon ID %d", daemonID)
		return nil, err
	}
	return subnets, err
}

// Fetches the subnet by prefix from the database.
func GetSubnetsByPrefix(dbi dbops.DBI, prefix string) ([]Subnet, error) {
	subnets := []Subnet{}
	err := dbi.Model(&subnets).
		Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("SharedNetwork").
		Relation("LocalSubnets.Daemon.App.AccessPoints").
		Where("subnet.prefix = ?", prefix).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting subnets with prefix %s", prefix)
		return nil, err
	}
	return subnets, err
}

// Fetches all subnets belonging to a given family. If the family is set to 0
// it fetches both IPv4 and IPv6 subnet.
func GetAllSubnets(dbi dbops.DBI, family int) ([]Subnet, error) {
	subnets := []Subnet{}
	q := dbi.Model(&subnets).
		Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("SharedNetwork").
		Relation("LocalSubnets.Daemon.App.AccessPoints").
		Relation("LocalSubnets.Daemon.App.Machine").
		OrderExpr("id ASC")

	// Let's be liberal and allow other values than 0 too. The only special
	// ones are 4 and 6.
	if family == 4 || family == 6 {
		q = q.Where("family(subnet.prefix) = ?", family)
	}
	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting all subnets for family %d", family)
		return nil, err
	}
	return subnets, err
}

// Fetches all global subnets, i.e., subnets that do not belong to shared
// networks. If the family is set to 0 it fetches both IPv4 and IPv6 subnet.
func GetGlobalSubnets(dbi dbops.DBI, family int) ([]Subnet, error) {
	subnets := []Subnet{}
	q := dbi.Model(&subnets).
		Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("address_pool.id ASC"), nil
		}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("LocalSubnets.Daemon.App.AccessPoints").
		OrderExpr("id ASC").
		Where("subnet.shared_network_id IS NULL")

	// Let's be liberal and allow other values than 0 too. The only special
	// ones are 4 and 6.
	if family == 4 || family == 6 {
		q = q.Where("family(subnet.prefix) = ?", family)
	}
	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting global (top-level) subnets for family %d", family)
		return nil, err
	}
	return subnets, nil
}

// Fetches a collection of subnets from the database. The offset and
// limit specify the beginning of the page and the maximum size of the
// page. The appID is used to filter subnets to those handled by the
// given application.  The family is used to filter by IPv4 (if 4) or
// IPv6 (if 6). For all other values of the family parameter both IPv4
// and IPv6 subnets are returned. The filterText can be used to match
// the subnet prefix or pool ranges. The nil value disables such
// filtering. sortField allows indicating sort column in database and
// sortDir allows selection the order of sorting. If sortField is
// empty then id is used for sorting.  in SortDirAny is used then ASC
// order is used. This function returns a collection of subnets, the
// total number of subnets and error.
func GetSubnetsByPage(dbi dbops.DBI, offset, limit, appID, family int64, filterText *string, sortField string, sortDir SortDirEnum) ([]Subnet, int64, error) {
	subnets := []Subnet{}
	q := dbi.Model(&subnets).Distinct()

	// When filtering by appID we also need the local_subnet table as it holds the
	// application identifier.
	if appID != 0 {
		q = q.Join("INNER JOIN local_subnet AS ls ON subnet.id = ls.subnet_id")
		q = q.Join("INNER JOIN daemon AS d ON ls.daemon_id = d.id")
	}
	// Pools are also required when trying to filter by text.
	if filterText != nil {
		q = q.Join("LEFT JOIN address_pool AS ap ON subnet.id = ap.subnet_id")
	}
	// Include pools, shared network the subnets belong to, local subnet info
	// and the associated apps in the results.
	q = q.Relation("AddressPools", func(q *orm.Query) (*orm.Query, error) {
		return q.Order("address_pool.id ASC"), nil
	}).
		Relation("PrefixPools", func(q *orm.Query) (*orm.Query, error) {
			return q.Order("prefix_pool.id ASC"), nil
		}).
		Relation("SharedNetwork").
		Relation("LocalSubnets.Daemon.App.AccessPoints").
		Relation("LocalSubnets.Daemon.App.Machine")

	// Let's be liberal and allow other values than 0 too. The only special
	// ones are 4 and 6.
	if family == 4 || family == 6 {
		q = q.Where("family(subnet.prefix) = ?", family)
	}

	// Filter by appID.
	if appID != 0 {
		q = q.Where("d.app_id = ?", appID)
	}

	// Quick filtering by subnet prefix, pool ranges or shared network name.
	if filterText != nil {
		// The combination of the concat and host functions reconstruct the textual
		// version of the pool range as specified in Kea, e.g. 192.0.2.10-192.0.2.20.
		// This allows for quick filtering by strings like: 2.10-192.0.
		q = q.WhereGroup(func(q *orm.Query) (*orm.Query, error) {
			q = q.WhereOr("text(subnet.prefix) LIKE ?", "%"+*filterText+"%").
				WhereOr("concat(host(ap.lower_bound), '-', host(ap.upper_bound)) LIKE ?", "%"+*filterText+"%").
				WhereOr("shared_network.name LIKE ?", "%"+*filterText+"%")
			return q, nil
		})
	}

	// prepare sorting expression, offset and limit
	ordExpr := prepareOrderExpr("subnet", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	// This returns the limited results plus the total number of records.
	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, 0, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting subnets by page")
	}
	return subnets, int64(total), err
}

// Get list of Subnets with LocalSubnets ordered by SharedNetworkID.
func GetSubnetsWithLocalSubnets(dbi dbops.DBI) ([]*Subnet, error) {
	subnets := []*Subnet{}
	q := dbi.Model(&subnets)
	// only selected columns are returned for performance reasons
	q = q.Column("id", "shared_network_id", "prefix")
	q = q.Relation("LocalSubnets")
	q = q.Order("shared_network_id ASC")

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrap(err, "problem getting all subnets")
		return nil, err
	}
	return subnets, nil
}

// Associates a daemon with the subnet having a specified ID and prefix
// in a transaction. Internally, the association is made via the local_subnet
// table which holds the information about the subnet from the given daemon
// perspective, local subnet id, statistics etc.
func addDaemonToSubnet(tx *pg.Tx, subnet *Subnet, daemon *Daemon) error {
	localSubnetID := int64(0)
	// If the prefix is available we should try to match the subnet prefix
	// with the app's configuration and retrieve the local subnet id from
	// there.
	if len(subnet.Prefix) > 0 {
		localSubnetID = daemon.GetLocalSubnetID(subnet.Prefix)
	}
	localSubnet := LocalSubnet{
		SubnetID:      subnet.ID,
		DaemonID:      daemon.ID,
		LocalSubnetID: localSubnetID,
	}
	// Try to insert. If such association already exists we could maybe do
	// nothing, but we do update instead to force setting the new value
	// of the local_subnet_id if it has changed.
	_, err := tx.Model(&localSubnet).
		Column("subnet_id").
		Column("daemon_id").
		Column("local_subnet_id").
		OnConflict("(daemon_id, subnet_id) DO UPDATE").
		Set("daemon_id = EXCLUDED.daemon_id").
		Set("local_subnet_id = EXCLUDED.local_subnet_id").
		Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem associating the daemon %d with the subnet %s",
			daemon.ID, subnet.Prefix)
	}
	return err
}

// Associates a daemon with the subnet having a specified ID and prefix.
// It begins a new transaction when dbi has a *pg.DB type or uses an existing
// transaction when dbi has a *pg.Tx type.
func AddDaemonToSubnet(dbi dbops.DBI, subnet *Subnet, daemon *Daemon) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addDaemonToSubnet(tx, subnet, daemon)
		})
	}
	return addDaemonToSubnet(dbi.(*pg.Tx), subnet, daemon)
}

// Dissociates a daemon from the subnet having a specified id.
// The first returned value indicates if any row was removed from the
// local_subnet table.
func DeleteDaemonFromSubnet(dbi dbops.DBI, subnetID int64, daemonID int64) (bool, error) {
	localSubnet := &LocalSubnet{
		DaemonID: daemonID,
		SubnetID: subnetID,
	}
	rows, err := dbi.Model(localSubnet).WherePK().Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting the daemon with ID %d from the subnet with %d",
			daemonID, subnetID)
		return false, err
	}
	return rows.RowsAffected() > 0, nil
}

// Dissociates a daemon from the subnets. The first returned value
// indicates if any row was removed from the local_subnet table.
func DeleteDaemonFromSubnets(dbi dbops.DBI, daemonID int64) (int64, error) {
	result, err := dbi.Model((*LocalSubnet)(nil)).
		Where("daemon_id = ?", daemonID).
		Delete()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrapf(err, "problem deleting daemon %d from subnets", daemonID)
		return 0, err
	}
	return int64(result.RowsAffected()), nil
}

// Finds and returns an app associated with a subnet having the specified id.
func (s *Subnet) GetApp(appID int64) *App {
	for _, s := range s.LocalSubnets {
		daemon := s.Daemon
		if daemon.App != nil && daemon.App.ID == appID {
			return daemon.App
		}
	}
	return nil
}

// Iterates over the provided slice of subnets and stores them in the database
// if they are not there yet. In addition, it associates the subnets with the
// specified Kea application. Returns a list of added subnets.
func commitSubnetsIntoDB(tx *pg.Tx, networkID int64, subnets []Subnet, daemon *Daemon) (addedSubnets []*Subnet, err error) {
	for i := range subnets {
		subnet := &subnets[i]
		if subnet.ID == 0 {
			subnet.SharedNetworkID = networkID
			err = AddSubnet(tx, subnet)
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to add detected subnet %s to the database",
					subnet.Prefix)
				return nil, err
			}
			addedSubnets = append(addedSubnets, subnet)
		}
		err = AddDaemonToSubnet(tx, subnet, daemon)
		if err != nil {
			err = pkgerrors.WithMessagef(err, "unable to associate detected subnet %s with Kea daemon of ID %d", subnet.Prefix, daemon.ID)
			return nil, err
		}

		err = CommitSubnetHostsIntoDB(tx, subnet, daemon, HostDataSourceConfig)
		if err != nil {
			return nil, err
		}
	}
	return addedSubnets, nil
}

// Iterates over the shared networks, subnets and hosts and commits them to the database.
// In addition it associates them with the specified app. Returns a list of added subnets.
// This function runs all database operations in a transaction.
func commitNetworksIntoDB(tx *pg.Tx, networks []SharedNetwork, subnets []Subnet, daemon *Daemon) ([]*Subnet, error) {
	var (
		addedSubnets      []*Subnet
		addedSubnetsToNet []*Subnet
		err               error
	)

	// Go over the networks that the Kea daemon belongs to.
	for i := range networks {
		network := &networks[i]
		if network.ID == 0 {
			// This is new shared network. Add it to the database.
			err = AddSharedNetwork(tx, network)
			if err != nil {
				err = pkgerrors.WithMessagef(err, "unable to add detected shared network %s to the database",
					network.Name)
				return nil, err
			}
		}
		// Associate subnets with the daemon.
		addedSubnetsToNet, err = commitSubnetsIntoDB(tx, network.ID, network.Subnets, daemon)
		if err != nil {
			return nil, err
		}
		addedSubnets = append(addedSubnets, addedSubnetsToNet...)
	}

	// Finally, add top level subnets to the database and associate them with
	// the Kea daemon.
	addedSubnetsToNet, err = commitSubnetsIntoDB(tx, 0, subnets, daemon)
	if err != nil {
		return nil, err
	}
	addedSubnets = append(addedSubnets, addedSubnetsToNet...)

	return addedSubnets, nil
}

// Iterates over the shared networks, subnets and hosts and commits them to the database.
// In addition it associates them with the specified daemon. Returns a list of added subnets.
func CommitNetworksIntoDB(dbi dbops.DBI, networks []SharedNetwork, subnets []Subnet, daemon *Daemon) (addedSubnets []*Subnet, err error) {
	if db, ok := dbi.(*pg.DB); ok {
		err = db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			addedSubnets, err = commitNetworksIntoDB(tx, networks, subnets, daemon)
			return err
		})
		return
	}
	addedSubnets, err = commitNetworksIntoDB(dbi.(*pg.Tx), networks, subnets, daemon)
	return
}

// Fetch all local subnets for indicated app.
func GetAppLocalSubnets(dbi dbops.DBI, appID int64) ([]*LocalSubnet, error) {
	subnets := []*LocalSubnet{}
	q := dbi.Model(&subnets)
	q = q.Join("INNER JOIN daemon AS d ON local_subnet.daemon_id = d.id")
	// only selected columns are returned while stats columns are skipped for performance reasons (they are pretty big json fields)
	q = q.Column("daemon_id", "subnet_id", "local_subnet_id")
	q = q.Relation("Subnet")
	q = q.Relation("Daemon.App")
	q = q.Where("d.app_id = ?", appID)

	err := q.Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting all local subnets for app %d", appID)
		return nil, err
	}
	return subnets, nil
}

// Update stats pulled for given local subnet.
func (lsn *LocalSubnet) UpdateStats(dbi dbops.DBI, stats SubnetStats) error {
	lsn.Stats = stats
	lsn.StatsCollectedAt = storkutil.UTCNow()
	q := dbi.Model(lsn)
	q = q.Column("stats", "stats_collected_at")
	q = q.WherePK()
	result, err := q.Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem updating stats in local subnet: [daemon:%d, subnet:%d, local subnet:%d]",
			lsn.DaemonID, lsn.SubnetID, lsn.LocalSubnetID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "local subnet: [daemon:%d, subnet:%d, local subnet:%d] does not exist",
			lsn.DaemonID, lsn.SubnetID, lsn.LocalSubnetID)
	}
	return err
}

// Update statistics in Subnet.
func (s *Subnet) UpdateStatistics(dbi dbops.DBI, statistics utilizationStats) error {
	addrUtilization := statistics.GetAddressUtilization()
	pdUtilization := statistics.GetDelegatedPrefixUtilization()
	s.AddrUtilization = int16(addrUtilization * 1000)
	s.PdUtilization = int16(pdUtilization * 1000)
	s.Stats = statistics.GetStatistics()
	s.StatsCollectedAt = time.Now().UTC()
	q := dbi.Model(s)
	q = q.Column("addr_utilization", "pd_utilization", "stats", "stats_collected_at")
	q = q.WherePK()
	result, err := q.Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem updating statistics in the subnet: %d",
			s.ID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "subnet with ID %d does not exist", s.ID)
	}
	return err
}

// Deletes subnets which are not associated with any apps. Returns deleted subnet
// count and an error.
func DeleteOrphanedSubnets(dbi dbops.DBI) (int64, error) {
	subquery := dbi.Model(&[]LocalSubnet{}).
		Column("id").
		Limit(1).
		Where("subnet.id = local_subnet.subnet_id")
	result, err := dbi.Model(&[]Subnet{}).
		Where("(?) IS NULL", subquery).
		Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting orphaned subnets")
		return 0, err
	}
	return int64(result.RowsAffected()), nil
}
