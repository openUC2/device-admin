import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
  show() {
    this.element.classList.remove('is-hidden');
  }
  hide() {
    this.element.classList.add('is-hidden');
  }
}
