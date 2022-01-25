import 'react-app-polyfill/stable';
import 'core-js';
import ReactDOM from 'react-dom';
import App from './App';

import { Provider } from 'react-redux'
import store from './state/store';
import {loadWebsocketToken} from './state/middleware/websocket';

store.dispatch(loadWebsocketToken)

ReactDOM.render(
  <Provider store={store}>
    <App/>
  </Provider>,
  document.getElementById('root')
);

