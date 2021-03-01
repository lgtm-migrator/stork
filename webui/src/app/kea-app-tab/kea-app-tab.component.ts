import { Component, OnInit, Input, Output, EventEmitter } from '@angular/core'
import { ActivatedRoute } from '@angular/router'
import { forkJoin } from 'rxjs'

import * as moment from 'moment-timezone'

import { MessageService, MenuItem } from 'primeng/api'

import { ServicesService } from '../backend/api/api'
import { ServerDataService } from '../server-data.service'

import {
    durationToString,
    daemonStatusErred,
    daemonStatusIconName,
    daemonStatusIconColor,
    daemonStatusIconTooltip,
} from '../utils'

@Component({
    selector: 'app-kea-app-tab',
    templateUrl: './kea-app-tab.component.html',
    styleUrls: ['./kea-app-tab.component.sass'],
})
export class KeaAppTabComponent implements OnInit {
    private _appTab: any
    @Output() refreshApp = new EventEmitter<number>()
    @Input() refreshedAppTab: any

    daemons: any[] = []

    activeTabIndex = 0

    /**
     * Holds a map of existing apps' names and ids.
     *
     * The apps' names are used in rename-app-dialog component to validate
     * the user input.
     */
    existingApps: any = []

    /**
     * Holds a set of existing machines' addresses.
     *
     * The machines' addresses are used in rename-app-dialog component to
     * validate the user input.
     */
    existingMachines: any = []

    /**
     * Controls whether the rename-app-dialog is visible or not.
     */
    appRenameDialogVisible = false

    /**
     * Indicates if a pencil icon was clicked.
     *
     * As a result of clicking this icon a dialog box is shown to
     * rename an app. Loading the dialog box may take a while before
     * the information about available apps and machines is loaded.
     * In the meantime, a spinner is shown, indicating that the dialog
     * box is loading.
     */
    showRenameDialogClicked = false

    /**
     * Event emitter sending an event to the parent component when an app is
     * renamed.
     */
    @Output() renameApp = new EventEmitter<string>()

    constructor(
        private route: ActivatedRoute,
        private servicesApi: ServicesService,
        private serverData: ServerDataService,
        private msgService: MessageService
    ) {}

    /**
     * Subscribes to the updates of the information about daemons
     *
     * The information about the daemons may be updated as a result of
     * pressing the refresh button in the app tab. In such case, this
     * component emits an event to which the parent component reacts
     * and updates the daemons. When the daemons are updated, it
     * notifies this compoment via the subscription mechanism.
     */
    ngOnInit() {
        this.refreshedAppTab.subscribe((data) => {
            if (data) {
                this.initDaemons(data.app.details.daemons)
            }
        })
    }

    /**
     * Selects new application tab
     *
     * As a result, the local information about the daemons is updated.
     *
     * @param appTab pointer to the new app tab data structure.
     */
    @Input()
    set appTab(appTab) {
        this._appTab = appTab
        // Refresh local information about the daemons presented by this
        // component.
        this.initDaemons(appTab.app.details.daemons)
    }

    /**
     * Returns information about currently selected app tab.
     */
    get appTab() {
        return this._appTab
    }

    /**
     * Initializes information about the daemons according to the information
     * carried in the provided parameter.
     *
     * As a result of invoking this function, the view of the component will be
     * updated.
     *
     * @param appTabDaemons information about the daemons stored in the app tab
     *                      data structure.
     */
    private initDaemons(appTabDaemons) {
        const activeDaemonTabName = this.route.snapshot.queryParams.daemon || null
        const daemonMap = []
        for (const d of appTabDaemons) {
            daemonMap[d.name] = d
        }
        const DMAP = [
            ['dhcp4', 'DHCPv4'],
            ['dhcp6', 'DHCPv6'],
            ['d2', 'DDNS'],
            ['ca', 'CA'],
            ['netconf', 'NETCONF'],
        ]
        const daemons = []
        let idx = 0
        for (const dm of DMAP) {
            if (daemonMap[dm[0]] !== undefined) {
                daemonMap[dm[0]].niceName = dm[1]
                daemonMap[dm[0]].subnets = []
                daemonMap[dm[0]].totalSubnets = 0
                daemonMap[dm[0]].statusErred = this.daemonStatusErred(daemonMap[dm[0]])
                daemons.push(daemonMap[dm[0]])

                if (dm[0] === activeDaemonTabName) {
                    this.activeTabIndex = idx
                }
                idx += 1
            }
        }
        this.daemons = daemons
    }

    /**
     * An action triggered when refresh button is pressed.
     */
    refreshAppState() {
        this.refreshApp.emit(this._appTab.app.id)
    }

    /**
     * Converts duration to pretty string.
     *
     * @param duration duration value to be converted.
     *
     * @returns duration as text
     */
    showDuration(duration) {
        return durationToString(duration)
    }

    /**
     * Returns boolean value indicating if there is an issue with communication
     * with the active daemon.
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @return true if there is a communication problem with the daemon,
     *         false otherwise.
     */
    private daemonStatusErred(daemon): boolean {
        if (daemon.active && daemonStatusErred(daemon)) {
            return true
        }
        return false
    }

    /**
     * Returns the name of the icon to be used when presenting daemon status
     *
     * The icon selected depends on whether the daemon is active or not
     * active and whether there is a communication with the daemon or
     * not.
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @returns ban icon if the daemon is not active, times icon if the daemon
     *          should be active but the communication with it is borken and
     *          check icon if the communication with the active daemon is ok.
     */
    daemonStatusIconName(daemon) {
        return daemonStatusIconName(daemon)
    }

    /**
     * Returns the color of the icon used when presenting daemon status
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @returns grey color if the daemon is not active, red if the daemon is
     *          active but there are communication issues, green if the
     *          communication with the active daemon is ok.
     */
    daemonStatusIconColor(daemon) {
        return daemonStatusIconColor(daemon)
    }

    /**
     * Returns error text to be displayed when there is a communication issue
     * with a given daemon
     *
     * @param daemon data structure holding the information about the daemon.
     *
     * @returns Error text. It includes hints about the communication
     *          problems when such problems occur, e.g. it includes the
     *          hint whether the communication is with the agent or daemon.
     */
    daemonStatusErrorText(daemon) {
        return daemonStatusIconTooltip(daemon)
    }

    changeMonitored(daemon) {
        const dmn = { monitored: !daemon.monitored }
        this.servicesApi.updateDaemon(daemon.id, dmn).subscribe(
            (data) => {
                daemon.monitored = dmn.monitored
            },
            (err) => {
                console.warn('failed to update monitoring flag in daemon')
            }
        )
    }

    /**
     * Checks if the specified log target can be viewed
     *
     * Only the logs that are stored in the file can be viewed in Stork. The
     * logs output to stdout, stderr or syslog can't be viewed in Stork.
     *
     * @param target log target output location
     * @returns true if the log target can be viewed, false otherwise.
     */
    logTargetViewable(target): boolean {
        return target !== 'stdout' && target !== 'stderr' && !target.startsWith('syslog')
    }

    /**
     * Reacts to submitting a new app name from the dialog.
     *
     * This function is called when a user presses the rename button in
     * the app-rename-app-dialog component. It attempts to submit the new
     * name to the server.
     *
     * If the app is successfully renamed, the app name is refreshed in
     * the app tab view. Additionally, the success message is displayed
     * in the message service.
     *
     * @param event holds new app name.
     */
    handleRenameDialogSubmitted(event) {
        this.servicesApi.renameApp(this.appTab.app.id, { name: event }).subscribe(
            (data) => {
                // Renaming the app was successful.
                this.msgService.add({
                    severity: 'success',
                    summary: 'App renamed',
                    detail: 'App successfully renamed to ' + event,
                })
                // Let's update the app name in the current tab.
                this.appTab.app.name = event
                // Notify the parent component about successfully renaming the app.
                this.renameApp.emit(event)
            },
            (err) => {
                // Renaming the app failed.
                let msg = err.statusText
                if (err.error && err.error.message) {
                    msg = err.error.message
                }
                this.msgService.add({
                    severity: 'error',
                    summary: 'App renaming erred',
                    detail: 'App renaming to ' + event + ' erred: ' + msg,
                    life: 10000,
                })
            }
        )
    }

    /**
     * Reacts to hiding a dialog box for renaming an app.
     *
     * This function is called when a dialog box for renaming an app is
     * closed. It is triggered both in the case when the form is submitted
     * or cancelled.
     */
    handleRenameDialogHidden() {
        this.appRenameDialogVisible = false
    }

    /**
     * Shows a dialog for renaming an app.
     *
     * The dialog box component requires a set of machines' addresses
     * and a map of existing apps' names to validate the new app name.
     * Therefore, this function attempts to load the machines' addresses
     * and apps' names prior to displaying the dialog. If it fails, the
     * dialog box is not displayed.
     */
    showRenameAppDialog() {
        this.showRenameDialogClicked = true
        forkJoin([this.serverData.getAppsNames(), this.serverData.getMachinesAddresses()]).subscribe(
            (data) => {
                this.existingApps = data[0]
                this.existingMachines = data[1]
                this.appRenameDialogVisible = true
                this.showRenameDialogClicked = false
            },
            (err) => {
                this.msgService.add({
                    severity: 'error',
                    summary: 'Fetching apps and machines failed',
                    detail: 'Fetching apps and machines list from the server failed',
                    life: 10000,
                })
                this.showRenameDialogClicked = false
            }
        )
    }
}
