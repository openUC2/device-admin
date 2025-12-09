import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
  static targets = ['checkable', 'textbox'];

  toggle() {
    if (this.checkableTarget.checked) {
      return;
    }
    this.textboxTarget.value = '';
  }

  edit() {
    this.checkableTarget.checked = true;
  }

  setPlaceholder(event) {
    this.textboxTarget.placeholder = event.params.placeholder;
  }
}
