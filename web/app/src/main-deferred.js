import { Application } from '@hotwired/stimulus';
import {
  CheckableTextboxController,
  DefaultScrollableController,
  DropdownTextboxController,
  EventController,
  FormSubmissionController,
  HideableController,
  NavigationLinkController,
  NavigationMenuController,
  PasswordInputController,
  ThemeController,
  Turbo,
  TurboCableStreamSourceElement,
  TurboCacheController,
  ShowableController,
  streamActionReload,
} from './sprinkles';

Turbo.session.drive = true;
Turbo.StreamActions.reload = streamActionReload;

customElements.define(
  'turbo-cable-stream-source',
  TurboCableStreamSourceElement,
);

const Stimulus = Application.start();
Stimulus.register('checkable-textbox', CheckableTextboxController);
Stimulus.register('default-scrollable', DefaultScrollableController);
Stimulus.register('dropdown-textbox', DropdownTextboxController);
Stimulus.register('event', EventController);
Stimulus.register('form-submission', FormSubmissionController);
Stimulus.register('hideable', HideableController);
Stimulus.register('navigation-link', NavigationLinkController);
Stimulus.register('navigation-menu', NavigationMenuController);
Stimulus.register('password-input', PasswordInputController);
Stimulus.register('theme', ThemeController);
Stimulus.register('turbo-cache', TurboCacheController);
Stimulus.register('showable', ShowableController);

if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/sw.js');
}

// Prevent noscript elements from being processed. Refer to
// https://discuss.hotwired.dev/t/turbo-processes-noscript-children-when-merging-head/2552
document.addEventListener('turbo:before-render', (event) => {
  for (var e of event.detail.newBody.querySelectorAll('noscript')) {
    e.remove();
  }
});
