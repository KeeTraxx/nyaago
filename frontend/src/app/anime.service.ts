import { Injectable } from '@angular/core';
import {Http, Response} from "@angular/http";
import {Observable} from "rxjs";
import {Show} from "./show";

@Injectable()
export class AnimeService {

  constructor(private http:Http) { }

  getShows():Observable<Show[]> {
    return this.http.get("/api/animes").map(res => res.json());
  }

  toggleShow(show, subbingGroup, resolution):Observable<Show> {
    return this.http.post("/api/animes/"+show.id+"/toggle", {
      subbing_group_id: subbingGroup.id,
      resolution
    }).map(res => res.json());
  }

  fetchShow(show:string) {
    return this.http.post("/api/fetch", show).map(res => res.json());
  }

}
