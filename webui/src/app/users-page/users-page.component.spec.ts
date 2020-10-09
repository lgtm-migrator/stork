import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { UsersPageComponent } from './users-page.component'
import { ActivatedRoute, Router } from '@angular/router'
import { FormBuilder } from '@angular/forms'
import { ServicesService, UsersService } from '../backend'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { MessageService } from 'primeng/api'
import { of } from 'rxjs'

class MockParamMap {
    get(name: string): string | null {
        return null
    }
}

describe('UsersPageComponent', () => {
    let component: UsersPageComponent
    let fixture: ComponentFixture<UsersPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [],
            declarations: [UsersPageComponent],
            providers: [
                FormBuilder,
                UsersService,
                HttpClient,
                HttpHandler,
                ServicesService,
                MessageService,
                {
                    provide: ActivatedRoute,
                    useValue: {
                        paramMap: of(new MockParamMap()),
                    },
                },
                {
                    provide: Router,
                    useValue: {},
                },
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(UsersPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
