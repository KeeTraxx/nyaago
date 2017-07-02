import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { NgModule } from '@angular/core';
import {FormsModule, ReactiveFormsModule} from '@angular/forms';
import { HttpModule } from '@angular/http';

import {AppComponent, DetailsDialog} from './app.component';
import {MaterialModule} from "@angular/material";
import {AnimeService} from "./anime.service";
import { TimeagoPipe } from './timeago.pipe';



@NgModule({
  declarations: [
    AppComponent,
    TimeagoPipe,
    DetailsDialog
  ],
  imports: [
    BrowserModule,
    FormsModule,
    HttpModule,
    ReactiveFormsModule,
    MaterialModule,
    BrowserAnimationsModule
  ],
  entryComponents: [
    DetailsDialog
  ],
  providers: [AnimeService],
  bootstrap: [AppComponent]
})
export class AppModule { }
