import { TestController } from '../actions';
import client from '../apiclient';


async function loadInitialState(dispatch) {
    const response = await client.get('initialState')
    dispatch({ type: 'TEST_CONTROLLER::INITIAL_STATE_LOADED', payload: response })
}

/**
 * initialStateMiddleware will monitor for the system being ready
 * and trigger loading the initial state
 * @param {*} storeAPI The redux store instance
 * @returns the result of calling next(action)
 */
const initialStateMiddleware = storeAPI => next => action => {
    if (action.type === TestController.SystemStateChanged) {
        if (action.payload?.state === "running") {
            storeAPI.dispatch(loadInitialState);
        }
    }
    // Do something in here, when each action is dispatched
    return next(action)
}

export default initialStateMiddleware;