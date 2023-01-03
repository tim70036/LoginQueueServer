import { check } from 'k6';
import { vu } from 'k6/execution';
import ws from 'k6/ws';

import {
  randomIntBetween,
  randomString,
  randomItem,
  uuidv4,
  findBetween,
} from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// not using SharedArray here will mean that the code in the function call (that is what loads and
// parses the json) will be executed per each VU which also means that there will be a complete copy
// per each VU
const userNum = 10000;
const wsHost = 'wss://login-queue-server.game-soul-swe.com:5487/ws';

export const options = {
  scenarios: {
    // one: {
    //   executor: 'per-vu-iterations',
    //   vus: userNum,
    //   iterations: 1,
    //   maxDuration: '10m',
    // },
    ramp: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '2m', target: userNum }, // ramp up
        { duration: '40m', target: userNum }, // stay
        { duration: '10s', target: 0 }, // scale down. (optional)
      ],
      gracefulRampDown: '10s',
    },
  },

  // For Unity client, 1 user has max 10 concurrent request (defined in addressable asset)
  batch: 10, // The maximum number of simultaneous/parallel connections in total that an http.batch() call in a VU can make.
  batchPerHost: 10, // The maximum number of simultaneous/parallel connections for the same hostname that an http.batch() call in a VU can make.

  discardResponseBodies: true, //  Lessens the amount of memory required and the amount of GC - reducing the load on the testing machine, and probably producing more reliable test results.

  noConnectionReuse: false, // Whether a connection is reused throughout different actions of the same virtual user and in the same iteration.
  noVUConnectionReuse: false, // Whether k6 should reuse TCP connections between iterations of a VU.
};

export default function () {
  // Setup
  const vuId = vu.idInTest - 1;
 
  const res = ws.connect(wsHost, { headers: { id: vuId, platform: 'Android' }}, function (socket) {
    socket.on('open', () => {
      console.log(`ws connected vuId[${vuId}]`);

      socket.send(JSON.stringify({
        eventCode: 1001,
        eventData: {
          type: 0,
          token: "asdz23asda-123sac"
        },
      }));
    });

    // socket.on('message', (data) => {
    //   console.log('Message received: ', data)
    // });
    // socket.on('ping',  () => console.log(`ws ping vuId[${vuId}]`)); // k6 will send pong for us.
    // socket.on('pong', () => console.log(`ws pong vuId[${vuId}]`));

    socket.on('close', () => console.log(`ws closed vuId[${vuId}]`));
    socket.on('error', function (e) {
      if (e.error() != 'websocket: close sent') {
        console.log(`ws error vuId[${vuId}]`, e.error());
      }
    });

    
    const sessionDuration = 6000000; // milisec
    // const sessionDuration = randomIntBetween(60000, 600000); // 1~10 min
    socket.setTimeout(function () {
        console.log(`ws closing vuId[${vuId}]`);
        socket.close();
    }, sessionDuration);
  });
  check(res, { 'status is 101': (r) => r && r.status === 101 });
}
