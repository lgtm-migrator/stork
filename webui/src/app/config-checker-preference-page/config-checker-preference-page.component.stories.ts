import { HttpClientModule } from '@angular/common/http'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { Meta, moduleMetadata, Story } from '@storybook/angular'
import { MessageService } from 'primeng/api'
import { ChipModule } from 'primeng/chip'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TableModule } from 'primeng/table'
import { ToastModule } from 'primeng/toast'
import { ConfigChecker, ConfigCheckerPreferences, ConfigCheckers, ConfigReports, ServicesService } from '../backend'
import { ConfigCheckerPreferencePickerComponent } from '../config-checker-preference-picker/config-checker-preference-picker.component'
import { ConfigCheckerPreferenceUpdaterComponent } from '../config-checker-preference-updater/config-checker-preference-updater.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { ConfigCheckerPreferencePageComponent } from './config-checker-preference-page.component'
import mockAddon from 'storybook-addon-mock'
import { toastDecorator } from '../utils.stories'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { RouterTestingModule } from '@angular/router/testing'
import { ButtonModule } from 'primeng/button'

const mockPreferencesData: ConfigCheckers = {
    items: [
        {
            name: 'reservations_out_of_pool',
            selectors: ['each-daemon', 'kea-daemon'],
            state: ConfigChecker.StateEnum.Disabled,
            triggers: ['manual', 'config change'],
            globallyEnabled: false,
        },
        {
            name: 'subnet_dispensable',
            selectors: ['each-daemon'],
            state: ConfigChecker.StateEnum.Enabled,
            triggers: ['manual', 'config change'],
            globallyEnabled: true,
        },
    ],
    total: 2,
}

export default {
    title: 'App/ConfigCheckerPreferencePage',
    component: ConfigCheckerPreferencePageComponent,
    decorators: [
        moduleMetadata({
            imports: [
                TableModule,
                ChipModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                HttpClientModule,
                ToastModule,
                BreadcrumbModule,
                RouterTestingModule,
                ButtonModule,
            ],
            declarations: [
                HelpTipComponent,
                ConfigCheckerPreferencePageComponent,
                ConfigCheckerPreferencePickerComponent,
                ConfigCheckerPreferenceUpdaterComponent,
                BreadcrumbsComponent,
            ],
            providers: [MessageService, ServicesService],
        }),
        mockAddon,
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/daemons/:daemonId/config-checkers',
                method: 'GET',
                status: 200,
                delay: 2000,
                response: mockPreferencesData,
            },
            {
                url: 'http://localhost/api/daemons/:daemonId/config-checkers',
                method: 'PUT',
                status: 200,
                response: (request) => {
                    const { body } = request
                    const preferences: ConfigCheckerPreferences = JSON.parse(body)

                    for (let preference of preferences.items) {
                        for (let checker of mockPreferencesData.items) {
                            if (preference.name === checker.name) {
                                checker.state = preference.state
                            }
                        }
                    }
                    return mockPreferencesData
                },
            },
            {
                url: 'http://localhost/api/daemons/:daemonId/config-reports?start=0&limit=5',
                method: 'GET',
                status: 200,
                delay: 2000,
                response: {
                    total: 2,
                    review: {
                        createdAt: '2022-08-25T12:34:56',
                        daemonId: 1,
                        id: 1,
                    },
                    items: [
                        {
                            checker: 'reservations_out_of_pool',
                            content: 'Something is wrong',
                            createdAt: '2022-08-25T12:34:56',
                            id: 1,
                        },
                        {
                            checker: 'subnet_dispensable',
                            content: 'Foobar',
                            createdAt: '2022-08-25T12:34:56',
                            id: 2,
                        },
                    ],
                } as ConfigReports,
            },
            {
                url: 'http://localhost/api/daemons/:daemonID/config-review',
                method: 'PUT',
                status: 200,
                delay: 1000,
            },
        ],
    },
} as Meta

const Template: Story<ConfigCheckerPreferencePageComponent> = (args: ConfigCheckerPreferencePageComponent) => ({
    props: args,
})

export const Primary = Template.bind({})
