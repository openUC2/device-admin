import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
  static targets = ['input', 'toggler'];

  connect() {
    // The toggler only works with Javascript, so we only show it if Javascript is enabled
    this.togglerTarget.classList.remove('is-hidden');
  }
  toggle() {
    console.log('toggling...');
    if (this.inputTarget.type === 'password') {
      this.inputTarget.type = 'text';
    } else {
      this.inputTarget.type = 'password';
    }
  }
}
