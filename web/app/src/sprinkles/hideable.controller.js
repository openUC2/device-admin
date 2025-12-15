import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
  static targets = ['hider'];

  connect() {
    // The hider only works with Javascript, so we only show it if Javascript is enabled
    this.hiderTarget.classList.remove('is-hidden');
  }

  hide(event) {
    this.element.classList.add('is-hidden');
  }
}
