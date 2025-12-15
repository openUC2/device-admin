import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
  static targets = ['submit', 'submitter'];

  submit() {
    if (this.hasSubmitTarget) {
      this.submitTarget.setAttribute('disabled', true);
    }
    const progress = document.createElement('progress');
    progress.classList.add('progress');
    progress.classList.add('is-small');
    progress.classList.add('is-info');
    if (this.hasSubmitterTarget) {
      this.submitterTarget.appendChild(progress);
    }
  }
}
