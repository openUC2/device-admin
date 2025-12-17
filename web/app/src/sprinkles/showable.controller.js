import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
  static targets = ['shower'];

  show() {
    this.showerTarget.classList.remove('is-hidden');
  }
  hide() {
    this.showerTarget.classList.add('is-hidden');
  }
}
