import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'
import { FormsModule } from '@angular/forms'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'

import { FieldsetModule } from 'primeng/fieldset'
import { ConfirmationService, MessageService } from 'primeng/api'
import { TableModule } from 'primeng/table'
import { ConfirmDialogModule } from 'primeng/confirmdialog'

import { of, throwError } from 'rxjs'

import { DHCPService } from '../backend'
import { HostTabComponent } from './host-tab.component'
import { RouterModule } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { IdentifierComponent } from '../identifier/identifier.component'
import { TreeModule } from 'primeng/tree'
import { DhcpOptionSetViewComponent } from '../dhcp-option-set-view/dhcp-option-set-view.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TagModule } from 'primeng/tag'

describe('HostTabComponent', () => {
    let component: HostTabComponent
    let fixture: ComponentFixture<HostTabComponent>
    let dhcpApi: DHCPService
    let msgService: MessageService
    let confirmService: ConfirmationService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [DHCPService, ConfirmationService, MessageService],
            imports: [
                ConfirmDialogModule,
                FieldsetModule,
                FormsModule,
                HttpClientTestingModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                TableModule,
                RouterModule,
                RouterTestingModule,
                ToggleButtonModule,
                TreeModule,
                TagModule,
            ],
            declarations: [DhcpOptionSetViewComponent, HelpTipComponent, HostTabComponent, IdentifierComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HostTabComponent)
        component = fixture.componentInstance
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        confirmService = fixture.debugElement.injector.get(ConfirmationService)
        msgService = fixture.debugElement.injector.get(MessageService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display host information', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
                {
                    idType: 'hw-address',
                    idHexValue: '51:52:53:54:55:56',
                },
            ],
            addressReservations: [
                {
                    address: '2001:db8:1::1',
                },
                {
                    address: '2001:db8:1::2',
                },
            ],
            prefixReservations: [
                {
                    address: '2001:db8:2::/64',
                },
                {
                    address: '2001:db8:3::/64',
                },
            ],
            hostname: 'mouse.example.org',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'config',
                },
                {
                    appId: 2,
                    appName: 'mouse',
                    dataSource: 'api',
                },
            ],
        }
        const fakeLeases: any = {}
        spyOn(dhcpApi, 'getLeases').and.returnValue(of(fakeLeases))
        component.host = host
        fixture.detectChanges()
        expect(dhcpApi.getLeases).toHaveBeenCalled()

        const titleSpan = fixture.debugElement.query(By.css('#tab-title-span'))
        expect(titleSpan).toBeTruthy()
        expect(titleSpan.nativeElement.innerText).toBe('[1] Host in subnet 2001:db8:1::/64')

        const addressReservationsFieldset = fixture.debugElement.query(By.css('#address-reservations-fieldset'))
        expect(addressReservationsFieldset).toBeTruthy()
        expect(addressReservationsFieldset.nativeElement.textContent).toContain('2001:db8:1::1')
        expect(addressReservationsFieldset.nativeElement.textContent).toContain('2001:db8:1::2')

        const prefixReservationsFieldset = fixture.debugElement.query(By.css('#prefix-reservations-fieldset'))
        expect(prefixReservationsFieldset).toBeTruthy()
        expect(prefixReservationsFieldset.nativeElement.textContent).toContain('2001:db8:2::/64')
        expect(prefixReservationsFieldset.nativeElement.textContent).toContain('2001:db8:3::/64')

        const nonIPReservationsFieldset = fixture.debugElement.query(By.css('#non-ip-reservations-fieldset'))
        expect(nonIPReservationsFieldset).toBeTruthy()
        expect(nonIPReservationsFieldset.nativeElement.textContent).toContain('mouse.example.org')

        const hostIdsFieldset = fixture.debugElement.query(By.css('#dhcp-identifiers-fieldset'))
        expect(hostIdsFieldset).toBeTruthy()
        expect(hostIdsFieldset.nativeElement.textContent).toContain('duid')
        expect(hostIdsFieldset.nativeElement.textContent).toContain('hw-address')
        // DUID should be converted to textual form.
        expect(hostIdsFieldset.nativeElement.textContent).toContain('QRST')
        // HW address should remain in hexadecimal form.
        expect(hostIdsFieldset.nativeElement.textContent).toContain('51:52:53:54:55:56')

        const appsFieldset = fixture.debugElement.query(By.css('#apps-fieldset'))
        expect(appsFieldset).toBeTruthy()

        const appLinks = appsFieldset.queryAll(By.css('a'))
        expect(appLinks.length).toBe(2)
        expect(appLinks[0].attributes.href).toBe('/apps/kea/1')
        expect(appLinks[1].attributes.href).toBe('/apps/kea/2')

        let configTag = appsFieldset.query(By.css('.cfg-srctag'))
        expect(configTag).toBeTruthy()
        expect(configTag.nativeElement.innerText).toBe('config')
        configTag = appsFieldset.query(By.css('.hostcmds-srctag'))
        expect(configTag).toBeTruthy()
        expect(configTag.nativeElement.innerText).toBe('host_cmds')
    })

    it('should display global host tab title', () => {
        const host = {
            id: 2,
            subnetId: 0,
        }
        component.host = host
        fixture.detectChanges()

        const titleSpan = fixture.debugElement.query(By.css('#tab-title-span'))
        expect(titleSpan).toBeTruthy()
        expect(titleSpan.nativeElement.innerText).toBe('[2] Global host')
    })

    it('should handle error while fetching host information', () => {
        const fakeLeases: any = {}
        spyOn(dhcpApi, 'getLeases').and.returnValue(throwError({ status: 404 }))
        spyOn(msgService, 'add')
        const host = {
            id: 1,
        }
        component.host = host
        expect(dhcpApi.getLeases).toHaveBeenCalled()
        expect(msgService.add).toHaveBeenCalled()
    })

    it('should display lease information', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '01:02:03:04',
                },
                {
                    idType: 'hw-address',
                    idHexValue: 'f1:f2:f3:f4:f5:f6',
                },
            ],
            addressReservations: [
                {
                    address: '2001:db8:1::1',
                },
                {
                    address: '2001:db8:1::2',
                },
            ],
            prefixReservations: [
                {
                    address: '2001:db8:2::/64',
                },
                {
                    address: '2001:db8:3::/64',
                },
            ],
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'config',
                },
                {
                    appId: 2,
                    appName: 'mouse',
                    dataSource: 'api',
                },
            ],
        }
        const fakeLeases: any = {
            items: [
                {
                    id: 1,
                    ipAddress: '2001:db8:1::1',
                    state: 0,
                    hwAddress: 'f1:f2:f3:f4:f5:f6',
                    subnetId: 1,
                    cltt: 1616149050,
                    validLifetime: 3600,
                },
                {
                    id: 2,
                    ipAddress: '2001:db8:2::',
                    prefixLength: 64,
                    state: 0,
                    duid: 'e1:e2:e3:e4:e5:e6',
                    subnetId: 1,
                    cltt: 1616149050,
                    validLifetime: 3600,
                },
            ],
            conflicts: [2],
            erredApps: [],
        }
        spyOn(dhcpApi, 'getLeases').and.returnValue(of(fakeLeases))
        component.host = host
        fixture.detectChanges()
        expect(dhcpApi.getLeases).toHaveBeenCalled()

        const addressReservationsFieldset = fixture.debugElement.query(By.css('#address-reservations-fieldset'))
        expect(addressReservationsFieldset).toBeTruthy()
        const addressReservationTable = addressReservationsFieldset.query(By.css('table'))
        expect(addressReservationTable).toBeTruthy()
        let addressReservationTrs = addressReservationTable.queryAll(By.css('tr'))
        expect(addressReservationTrs.length).toBe(2)
        expect(addressReservationTrs[0].nativeElement.textContent).toContain('in use')
        expect(addressReservationTrs[1].nativeElement.textContent).toContain('unused')

        let links = addressReservationTrs[0].queryAll(By.css('a'))
        expect(links.length).toBe(1)
        expect(links[0].attributes.href).toBe('/dhcp/leases?text=2001:db8:1::1')
        expect(links[0].properties.text).toBe('2001:db8:1::1')

        const expandAddressLink = addressReservationTrs[0].query(By.css('button'))
        expect(expandAddressLink).toBeTruthy()
        expandAddressLink.nativeElement.click()
        fixture.detectChanges()

        addressReservationTrs = addressReservationTable.queryAll(By.css('tr'))
        expect(addressReservationTrs.length).toBe(3)
        expect(addressReservationTrs[1].nativeElement.textContent).toContain(
            'Found 1 assigned lease with the expiration time at '
        )

        const prefixReservationsFieldset = fixture.debugElement.query(By.css('#prefix-reservations-fieldset'))
        expect(prefixReservationsFieldset).toBeTruthy()
        const prefixReservationTable = prefixReservationsFieldset.query(By.css('table'))
        expect(prefixReservationTable).toBeTruthy()
        const prefixReservationTrs = prefixReservationTable.queryAll(By.css('tr'))
        expect(prefixReservationTrs.length).toBe(2)
        expect(prefixReservationTrs[0].nativeElement.textContent).toContain('in conflict')
        expect(prefixReservationTrs[1].nativeElement.textContent).toContain('unused')

        links = prefixReservationTrs[0].queryAll(By.css('a'))
        expect(links.length).toBe(1)
        expect(links[0].attributes.href).toBe('/dhcp/leases?text=2001:db8:2::')
        expect(links[0].properties.text).toBe('2001:db8:2::/64')
    })

    it('should display multiple lease information', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'hw-address',
                    idHexValue: 'f1:f2:f3:f4:f5:f6',
                },
            ],
            addressReservations: [
                {
                    address: '192.0.2.1',
                },
            ],
            subnetId: 1,
            subnetPrefix: '192.0.2.0/24',
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'config',
                },
                {
                    appId: 2,
                    appName: 'mouse',
                    dataSource: 'api',
                },
            ],
        }

        const fakeLeases: any = {
            items: [
                {
                    id: 1,
                    ipAddress: '192.0.2.1',
                    state: 0,
                    hwAddress: 'f1:f2:f3:f4:f5:f6',
                    subnetId: 1,
                    cltt: 1616149050,
                    validLifetime: 3600,
                },
                {
                    id: 2,
                    ipAddress: '192.0.2.1',
                    state: 0,
                    hwAddress: 'f1:f2:f3:f4:f5:f6',
                    subnetId: 1,
                    cltt: 1616149050,
                    validLifetime: 3600,
                },
            ],
            conflicts: [],
            erredApps: [],
        }
        const spy = spyOn(dhcpApi, 'getLeases')

        spy.and.returnValue(of(fakeLeases))
        component.host = host
        fixture.detectChanges()
        expect(dhcpApi.getLeases).toHaveBeenCalled()

        let addressReservationsFieldset = fixture.debugElement.query(By.css('#address-reservations-fieldset'))
        expect(addressReservationsFieldset).toBeTruthy()
        let addressReservationTable = addressReservationsFieldset.query(By.css('table'))
        expect(addressReservationTable).toBeTruthy()
        let addressReservationTrs = addressReservationTable.queryAll(By.css('tr'))
        expect(addressReservationTrs.length).toBe(1)
        expect(addressReservationTrs[0].nativeElement.textContent).toContain('in use')

        // Simulate the case that conflicted lease is returned. Note that here
        // we also simulate different order of leases.
        fakeLeases.items[1] = fakeLeases.items[0]
        fakeLeases.items[0] = {
            id: 2,
            ipAddress: '192.0.2.1',
            state: 0,
            hwAddress: 'e1:e2:e3:e4:e5:e6',
            subnetId: 1,
            cltt: 1616149050,
            validLifetime: 3600,
        }
        fakeLeases.conflicts.push(2)
        spy.and.returnValue(of(fakeLeases))
        component.refreshLeases()
        expect(dhcpApi.getLeases).toHaveBeenCalled()
        fixture.detectChanges()

        addressReservationsFieldset = fixture.debugElement.query(By.css('#address-reservations-fieldset'))
        expect(addressReservationsFieldset).toBeTruthy()
        addressReservationTable = addressReservationsFieldset.query(By.css('table'))
        expect(addressReservationTable).toBeTruthy()
        addressReservationTrs = addressReservationTable.queryAll(By.css('tr'))
        expect(addressReservationTrs.length).toBe(1)
        expect(addressReservationTrs[0].nativeElement.textContent).toContain('in conflict')
    })

    it('should return correct lease summary', () => {
        // Single lease in use.
        let leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Used,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                },
            ],
        }
        let summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found 1 assigned lease with the expiration time at')

        // Two leases in use.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Used,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                },
                {
                    hwAddress: '2a:2b:2c:2d:2e:2f',
                    cltt: 1000,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found 2 assigned leases with the latest expiration time at')

        // Single expired lease.
        // Set cltt so that the expiration time elapses 10 or more seconds ago.
        const testCltt = new Date().getTime() / 1000 - 3610
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: testCltt,
                validLifetime: 3600,
            },
            usage: component.Usage.Expired,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: testCltt,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toMatch(
            /Found 1 lease for this reservation that expired at \d{4}-\d{2}-\d{2}\s\d{2}\:\d{2}\:\d{2} \(\d{2} s ago\)/
        )

        // Two expired leases.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Expired,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                },
                {
                    hwAddress: '2a:2b:2c:2d:2e:2f',
                    cltt: 1000,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found 2 leases for this reservation. This includes a lease that expired at')

        // Single declined lease.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Declined,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found 1 lease for this reservation which is declined and has an expiration time at')

        // Two declined leases.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Declined,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                },
                {
                    hwAddress: '2a:2b:2c:2d:2e:2f',
                    cltt: 1000,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain(
            'Found 2 leases for this reservation. This includes a declined lease with expiration time at'
        )

        // Single conflicted lease with MAC address.
        leaseInfo = {
            culprit: {
                hwAddress: '1a:1b:1c:1d:1e:1f',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Conflicted,
            leases: [
                {
                    hwAddress: '1a:1b:1c:1d:1e:1f',
                    cltt: 0,
                    validLifetime: 3600,
                },
                {
                    hwAddress: '2a:2b:2c:2d:2e:2f',
                    cltt: 1000,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo)
        expect(summary).toContain('Found a lease with an expiration time at')
        expect(summary).toContain(
            'assigned to the client with MAC address=1a:1b:1c:1d:1e:1f, for which it was not reserved.'
        )

        // Conflicted lease with DUID.
        const leaseInfo2 = {
            culprit: {
                duid: '11:12:13',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Conflicted,
            leases: [
                {
                    duid: '11:12:13',
                    cltt: 0,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo2)
        expect(summary).toContain('Found a lease with an expiration time at')
        expect(summary).toContain('assigned to the client with DUID=11:12:13, for which it was not reserved.')

        // Conflicted lease with client-id.
        const leaseInfo3 = {
            culprit: {
                clientId: '11:12:13',
                cltt: 0,
                validLifetime: 3600,
            },
            usage: component.Usage.Conflicted,
            leases: [
                {
                    clientId: '11:12:13',
                    cltt: 0,
                    validLifetime: 3600,
                },
            ],
        }
        summary = component.getLeaseSummary(leaseInfo3)
        expect(summary).toContain('Found a lease with an expiration time at')
        expect(summary).toContain('assigned to the client with client-id=11:12:13, for which it was not reserved.')
    })

    it('should display host delete button for host reservation received over host_cmds', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: 'mouse.example.org',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'api',
                },
            ],
        }
        component.host = host
        fixture.detectChanges()
        const deleteBtn = fixture.debugElement.query(By.css('[label=Delete]'))
        expect(deleteBtn).toBeTruthy()

        // Simulate clicking on the button and make sure that the confirm dialog
        // has been displayed.
        spyOn(confirmService, 'confirm')
        deleteBtn.nativeElement.click()
        expect(confirmService.confirm).toHaveBeenCalled()
    })

    it('should emit an event indicating successful host deletion', fakeAsync(() => {
        const successResp: any = {}
        spyOn(dhcpApi, 'deleteHost').and.returnValue(of(successResp))
        spyOn(msgService, 'add')
        spyOn(component.hostDelete, 'emit')

        // Delete the host.
        component.host = {
            id: 1,
        }
        component.deleteHost()
        tick()
        // Success message should be displayed.
        expect(msgService.add).toHaveBeenCalled()
        // An event should be called.
        expect(component.hostDelete.emit).toHaveBeenCalledWith(component.host)
        // This flag should be cleared.
        expect(component.hostDeleted).toBeFalse()
    }))

    it('should not emit an event when host deletion fails', fakeAsync(() => {
        spyOn(dhcpApi, 'deleteHost').and.returnValue(throwError({ status: 404 }))
        spyOn(msgService, 'add')
        spyOn(component.hostDelete, 'emit')

        // Delete the host and receive an error.
        component.host = {
            id: 1,
        }
        component.deleteHost()
        tick()
        // Error message should be displayed.
        expect(msgService.add).toHaveBeenCalled()
        // The event shouldn't be emitted on error.
        expect(component.hostDelete.emit).not.toHaveBeenCalledWith(component.host)
        // This flag should be cleared.
        expect(component.hostDeleted).toBeFalse()
    }))

    it('should not display host delete button for host reservation from the config file', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: 'mouse.example.org',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    appName: 'frog',
                    dataSource: 'config',
                },
            ],
        }
        component.host = host
        fixture.detectChanges()
        // Unable to delete hosts specified in the config file.
        expect(fixture.debugElement.query(By.css('[label=Delete]'))).toBeFalsy()
    })

    it('should display different DHCP options for different servers separately', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: '',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    appName: 'frog',
                    dataSource: 'api',
                    options: [
                        {
                            code: 1024,
                        },
                        {
                            code: 1025,
                        },
                    ],
                    optionsHash: '1111',
                },
                {
                    appId: 2,
                    daemonId: 1,
                    appName: 'lion',
                    dataSource: 'api',
                    options: [
                        {
                            code: 1024,
                        },
                        {
                            code: 1026,
                        },
                    ],
                    optionsHash: '2222',
                },
            ],
        }
        component.host = host
        fixture.detectChanges()

        let fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(4)

        expect(fieldsets[2].properties.innerText).toContain('DHCP Options')
        expect(fieldsets[3].properties.innerText).toContain('DHCP Options')

        let frogLink = fieldsets[2].query(By.css('a'))
        expect(frogLink).toBeTruthy()
        expect(frogLink.properties.innerText).toContain('frog')
        expect(frogLink.properties.pathname).toBe('/apps/kea/1')

        let lionLink = fieldsets[3].query(By.css('a'))
        expect(lionLink).toBeTruthy()
        expect(lionLink.properties.innerText).toContain('lion')
        expect(lionLink.properties.pathname).toBe('/apps/kea/2')
    })

    it('should display the same DHCP options for different servers in one panel', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: '',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    appName: 'frog',
                    dataSource: 'api',
                    options: [
                        {
                            code: 1024,
                        },
                        {
                            code: 1025,
                        },
                    ],
                    optionsHash: '1111',
                },
                {
                    appId: 2,
                    daemonId: 1,
                    appName: 'lion',
                    dataSource: 'api',
                    options: [
                        {
                            code: 1024,
                        },
                        {
                            code: 1025,
                        },
                    ],
                    optionsHash: '1111',
                },
            ],
        }
        component.host = host
        fixture.detectChanges()

        let fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(3)

        expect(fieldsets[2].properties.innerText).toContain('DHCP Options')
        expect(fieldsets[2].properties.innerText).toContain('All Servers')
    })

    it('should display DHCP options panel for host with one daemon and include server name', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: '',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    appName: 'frog',
                    dataSource: 'api',
                    options: [
                        {
                            code: 1024,
                        },
                        {
                            code: 1025,
                        },
                    ],
                    optionsHash: '1111',
                },
            ],
        }
        component.host = host
        fixture.detectChanges()

        let fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(3)

        expect(fieldsets[2].properties.innerText).toContain('DHCP Options')

        let frogLink = fieldsets[2].query(By.css('a'))
        expect(frogLink).toBeTruthy()
        expect(frogLink.properties.innerText).toContain('frog')
        expect(frogLink.properties.pathname).toBe('/apps/kea/1')
    })

    it('should display a message about no DHCP options configured', () => {
        const host = {
            id: 1,
            hostIdentifiers: [
                {
                    idType: 'duid',
                    idHexValue: '51:52:53:54',
                },
            ],
            addressReservations: [],
            prefixReservations: [],
            hostname: '',
            subnetId: 1,
            subnetPrefix: '2001:db8:1::/64',
            localHosts: [
                {
                    appId: 1,
                    daemonId: 1,
                    appName: 'frog',
                    dataSource: 'api',
                },
                {
                    appId: 2,
                    daemonId: 1,
                    appName: 'lion',
                    dataSource: 'api',
                },
            ],
        }
        component.host = host
        fixture.detectChanges()

        let fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(3)
        expect(fieldsets[2].properties.innerText).toContain('No options configured.')
    })
})
