import { Controller } from '@hotwired/stimulus';

export default class extends Controller {
  static targets = ['datalist', 'dropdown', 'select', 'textbox', 'input'];

  connect() {
    this.setTextboxVisibility = (visible) => {
      if (visible) {
        this.textboxTarget.classList.remove('is-hidden');
        this.inputTarget.required = true;
        return;
      }
      this.textboxTarget.classList.add('is-hidden');
      this.inputTarget.required = false;
    };

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
        if (newOption.value === previousValue && document.activeElement !== this.inputTarget) {
          newOption.selected = true;
          optionSelected = true;
          this.setTextboxVisibility(false);
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
        this.setTextboxVisibility(true);
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
        this.setTextboxVisibility(true);
        this.inputTarget.focus();
        break;
      }

      this.setTextboxVisibility(false);
      break;
    }
  }
}
