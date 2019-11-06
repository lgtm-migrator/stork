import { Component, OnInit } from '@angular/core'
import { Router } from '@angular/router'

import { MenubarModule } from 'primeng/menubar'
import { MenuItem } from 'primeng/api'

import { AuthService, User } from './auth.service'

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.sass'],
})
export class AppComponent implements OnInit {
    title = 'Stork'
    currentUser = null

    menuItems: MenuItem[]

    constructor(private router: Router, private auth: AuthService) {
        this.auth.currentUser.subscribe(x => (this.currentUser = x))
    }

    ngOnInit() {
        this.menuItems = [
            {
                //     label: 'DHCP',
                // }, {
                //     label: 'DNS',
                // }, {
                label: 'Services',
                items: [
                    {
                        //     label: 'Kea DHCP'
                        // }, {
                        //     label: 'BIND DNS'
                        // }, {
                        label: 'Machines',
                        icon: 'fa fa-server',
                        routerLink: '/machines',
                    },
                ],
            },
        ]
    }

    signOut() {
        this.auth.logout()
        this.router.navigate(['/login'])
    }
}
