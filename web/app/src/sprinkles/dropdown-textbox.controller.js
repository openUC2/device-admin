import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
  static targets = ['input', 'datalist', 'dropdown', 'select'];

  connect() {
    // The dropdown only works with Javascript, so we only show it if Javascript is enabled
    this.dropdownTarget.classList.remove('is-hidden');
    this.inputTarget.setAttribute('list', '');

    this.updateDropdown = () => {
      this.selectTarget.innerHTML = '';
      const emptyOption = document.createElement('option');
      this.selectTarget.add(emptyOption);
      for (const option of this.datalistTarget.options) {
        const newOption = document.createElement('option');
        newOption.value = option.value;
        newOption.setAttribute('label', option.value); // TODO: only set label when select is opened
        this.selectTarget.add(newOption);
      }
    }
    this.updateDropdown()
  }

  updateDropdown() {
    this.updateDropdown()
  }

  select() {
    for (const option of this.selectTarget.options) {
      if (!option.selected) {
        continue;
      }
      if (option.value === '') {
        break;
      }

      this.inputTarget.value = option.value;
      option.selected = false;
      break; // TODO: maybe instead remove the label but restore it when the select is opened
    }
    this.selectTarget.options[0].selected = true
  }
}
