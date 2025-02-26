package dbmodel

import (
	"github.com/go-pg/pg/v10"
	"github.com/pkg/errors"
)

// Metric values calculated for specific subnet or shared network.
type CalculatedNetworkMetrics struct {
	// Subnet prefix or shared network name.
	Label string
	// Address utilization in percentage multiplied by 10.
	AddrUtilization int16
	// Delegated prefix utilization in percentage multiplied by 10.
	PdUtilization int16
}

// Metric values calculated from the database.
type CalculatedMetrics struct {
	AuthorizedMachines   int64
	UnauthorizedMachines int64
	UnreachableMachines  int64
	SubnetMetrics        []CalculatedNetworkMetrics
	SharedNetworkMetrics []CalculatedNetworkMetrics
}

// Calculates various metrics using several SELECT queries.
func GetCalculatedMetrics(db *pg.DB) (*CalculatedMetrics, error) {
	metrics := CalculatedMetrics{}
	err := db.Model().
		Table("machine").
		ColumnExpr("COUNT(*) FILTER (WHERE machine.authorized) AS \"authorized_machines\"").
		ColumnExpr("COUNT(*) FILTER (WHERE NOT(machine.authorized)) AS \"unauthorized_machines\"").
		ColumnExpr("COUNT(*) FILTER (WHERE machine.error IS NOT NULL) AS \"unreachable_machines\"").
		Select(&metrics)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot calculate global metrics")
	}

	err = db.Model().
		Table("subnet").
		ColumnExpr("\"prefix\" AS \"label\"").
		Column("addr_utilization", "pd_utilization").
		Select(&metrics.SubnetMetrics)

	if err != nil {
		return nil, errors.Wrap(err, "Cannot calculate subnet metrics")
	}

	err = db.Model().
		Table("shared_network").
		ColumnExpr("\"name\" AS \"label\"").
		Column("addr_utilization", "pd_utilization").
		Select(&metrics.SharedNetworkMetrics)

	if err != nil {
		return nil, errors.Wrap(err, "Cannot calculate shared network metrics")
	}

	return &metrics, nil
}
