import {Component, AfterViewInit, Inject} from '@angular/core';
import {AnimeService} from "./anime.service";
import {Show} from "./show";
import {Observable} from "rxjs";
import {MdDialog, MD_DIALOG_DATA} from "@angular/material";

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent implements AfterViewInit {

  private shows:Observable<Show[]>;

  constructor(private animeService: AnimeService, private dialog:MdDialog) {
  }


  ngAfterViewInit(): void {
    console.log('get shows!');
    this.shows = this.animeService.getShows();
  }

  isAutoDownloading(show, subbingGroup): boolean {
    return show.auto_download_group_id == subbingGroup.id;
  }

  toggleShow(show, subbingGroup, resolution) {
    this.animeService.toggleShow(show, subbingGroup, resolution).subscribe(s => {
      console.log(s);
      show.auto_download_resolution = s.auto_download_resolution;
      show.auto_download_group_id = s.auto_download_group_id;
    });
  }

  match(show, filter) {
    return show.name.match(new RegExp(filter, 'i'));
  }

  fetchShow(show:string) {
    this.animeService.fetchShow(show).subscribe(s => {
      console.log(s);
    });
  }

  showDetails(show) {
    console.log(show);
    this.dialog.open(DetailsDialog,{data: show});
  }

}

@Component({
  selector: 'dialog-overview-example-dialog',
  templateUrl: './details.dialog.html',
  styleUrls: ['./details.dialog.scss']
})
export class DetailsDialog {
  constructor(@Inject(MD_DIALOG_DATA) public data: any) { }
}
