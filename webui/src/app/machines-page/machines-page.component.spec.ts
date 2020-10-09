import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { MachinesPageComponent } from './machines-page.component'
import { ActivatedRoute, Router } from '@angular/router'
import { ServicesService, UsersService } from '../backend'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { MessageService } from 'primeng/api'
import { of } from 'rxjs'

describe('MachinesPageComponent', () => {
    let component: MachinesPageComponent
    let fixture: ComponentFixture<MachinesPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [
                MessageService,
                {
                    provide: ActivatedRoute,
                    useValue: {
                        paramMap: of({}),
                    },
                },
                {
                    provide: Router,
                    useValue: {},
                },
                ServicesService,
                HttpClient,
                HttpHandler,
                UsersService,
            ],
            declarations: [MachinesPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(MachinesPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
