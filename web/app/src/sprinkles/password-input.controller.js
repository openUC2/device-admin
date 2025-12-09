import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
  static targets = ['addons', 'input', 'toggler'];

  connect() {
    // The toggler only works with Javascript, so we only show it if Javascript is enabled
    this.togglerTarget.classList.remove('is-hidden');

    this.togglerTarget.disabled = this.inputTarget.value === '';

    if (this.addonsTarget) {
      this.addonsTarget.classList.add('has-addons');
    }
  }

  edit() {
    this.togglerTarget.disabled = this.inputTarget.value === '';
  }

  toggle() {
    if (this.inputTarget.type === 'password') {
      this.inputTarget.type = 'text';
    } else {
      this.inputTarget.type = 'password';
    }
  }
}
