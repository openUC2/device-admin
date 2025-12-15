import { createConsumer } from '@anycable/web';
import { MsgpackEncoder } from '@anycable/msgpack-encoder';

let consumers = {};

export function getActionCableConsumer(url, subprotocol) {
  if (subprotocol == undefined) {
    subprotocol = 'actioncable-v1-json';
  }

  let opts;
  switch (subprotocol) {
    default:
      throw `unsupported websocket subprotocol ${subprotocol}`;
    case 'actioncable-v1-json':
      break;
    case 'actioncable-v1-msgpack':
      opts = {
        protocol: subprotocol,
        encoder: new MsgpackEncoder(),
      };
      break;
  }

  if (consumers[url] === undefined) {
    consumers[url] = {};
  }
  if (consumers[url][subprotocol] == undefined) {
    consumers[url][subprotocol] = createConsumer(
      url === null ? undefined : url,
      opts,
    );
  }
  return consumers[url][subprotocol];
}

// Copied verbatim from the MIT-licensed absoluteWSUrl function at
// https://github.com/anycable/anycable-client/blob/master/packages/web/index.js
export function makeWebSocketURL(path) {
  if (path.match(/wss?:\/\//)) return path;

  if (typeof window !== 'undefined') {
    let proto = window.location.protocol.replace('http', 'ws');

    return `${proto}//${window.location.host}${path}`;
  }

  return path;
}
