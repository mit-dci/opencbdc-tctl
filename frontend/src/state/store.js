import { configureStore } from '@reduxjs/toolkit'

import reduxWebsocket from '@giantmachines/redux-websocket';
import testControllerMiddleware from './middleware';
import thunkMiddleware from 'redux-thunk';

import {commits, testruns, architectures, users, agents, system} from './slices';

const reduxWebsocketMiddleware = reduxWebsocket({deserializer:(r) => JSON.parse(r)});

const store = configureStore({
  reducer: {
    testruns: testruns.reducer,
    architectures: architectures.reducer,
    users: users.reducer,
    agents: agents.reducer,
    system: system.reducer,
    commits: commits.reducer,
  },
  middleware: [
    thunkMiddleware,
    reduxWebsocketMiddleware,
    ...testControllerMiddleware
  ],
  devTools: process.env.NODE_ENV !== 'production',
});

export default store
