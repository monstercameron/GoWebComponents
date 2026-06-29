import { bootstrapApplication } from "@angular/platform-browser";
import { provideZonelessChangeDetection } from "@angular/core";
import { AppComponent } from "./app/app.component";

// Zoneless change detection (stable in Angular 20): the app is signals-only, so it
// needs no zone.js — change detection is driven by signal reads in the template.
bootstrapApplication(AppComponent, {
  providers: [provideZonelessChangeDetection()],
}).catch((err) => console.error(err));
