import { Turbo } from './turbo';
import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
  clear(e) {
    // This assumes the controller is attached to a form!
    e.preventDefault();
    Turbo.clearCache();
    e.target.submit();
  }
}
