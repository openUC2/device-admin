// This is adapted from the MIT-licensed library @hotwired/turbo-rails.
import { connectStreamSource, disconnectStreamSource } from '@hotwired/turbo';
import { getActionCableConsumer, makeWebSocketURL } from './action-cable';

export default class TurboCableStreamSourceElement extends HTMLElement {
  async connectedCallback() {
    connectStreamSource(this);
    if (document.documentElement.hasAttribute('data-turbo-preview')) {
      return;
    }

    // Initialize channel
    const channel = {
      channel: this.getAttribute('channel'),
      name: this.getAttribute('name'),
      integrity: this.getAttribute('integrity'),
    };

    // Subscribe
    const consumer = getActionCableConsumer(
      makeWebSocketURL(this.getAttribute('cable-route')),
      this.getAttribute('websocket-subprotocol'),
    );
    this.subscription = consumer.subscriptions.create(channel, {
      received: this.dispatchMessageEvent.bind(this),
    });
  }

  disconnectedCallback() {
    disconnectStreamSource(this);
    if (this.subscription) this.subscription.unsubscribe();
  }

  dispatchMessageEvent(data) {
    return this.dispatchEvent(new MessageEvent('message', { data }));
  }
}
