import React from 'react';
import { useSelector } from 'react-redux';
import { BrowserRouter, Route, Switch } from 'react-router-dom';
import './scss/style.scss';

const loading = (
  <div className="pt-3 text-center">
    <div className="sk-spinner sk-spinner-pulse"></div>
  </div>
)

// Containers
const TheLayout = React.lazy(() => import('./containers/TheLayout'));

const App = () => {
  const systemState = useSelector(state => state?.system?.systemState);
  return <BrowserRouter>
    <React.Suspense fallback={loading}>
      {systemState === "running" && <Switch>
        <Route path="/" render={props => <TheLayout />} />
      </Switch>}
      {systemState !== "running" && <div>The controller is starting up, please wait...</div>}
    </React.Suspense>
  </BrowserRouter>
}

export default App;
