import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
  static targets = ['datalist', 'dropdown', 'select', 'textbox', 'input'];

  connect() {
    // The select is only populated via Javascript, so we only show it if Javascript is enabled
    this.dropdownTarget.classList.remove('is-hidden');

    this.updateSelect = () => {
      var previousValue = '';
      for (const option of this.selectTarget.options) {
        if (!option.selected || option.value === '') {
          continue;
        }
        previousValue = option.value;
        break;
      }
      if (previousValue === '') {
        previousValue = this.inputTarget.value;
      }

      // Add options from the datalist
      this.selectTarget.innerHTML = '';
      var optionSelected = false;
      for (const option of this.datalistTarget.options) {
        const newOption = document.createElement('option');
        newOption.value = option.value;
        newOption.setAttribute('label', option.value);
        if (newOption.value === previousValue) {
          newOption.selected = true;
          optionSelected = true;
          this.textboxTarget.classList.add('is-hidden');
        }
        this.selectTarget.add(newOption);
      }

      // Add the "other" option
      const separatorOption = document.createElement('option');
      separatorOption.setAttribute('label', '————');
      separatorOption.disabled = true;
      this.selectTarget.add(separatorOption);
      const emptyOption = document.createElement('option');
      emptyOption.setAttribute('label', '(other: specify below)');
      if (!optionSelected) {
        emptyOption.selected = true;
        this.textboxTarget.classList.remove('is-hidden');
        this.inputTarget.value = previousValue;
      }
      this.selectTarget.add(emptyOption);
    };
    this.updateSelect();
  }

  updateSelect() {
    this.updateSelect();
  }

  select() {
    for (const option of this.selectTarget.options) {
      if (!option.selected) {
        continue;
      }
      if (option.value === '') {
        this.textboxTarget.classList.remove('is-hidden');
        break;
      }

      this.textboxTarget.classList.add('is-hidden');
      break;
    }
  }
}
