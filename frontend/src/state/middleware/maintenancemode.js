import { TestController } from '../actions';

/**
 * maintenanceModeMiddleware will monitor for MaintenanceMode changes
 * and notify the user when they occur.
 * @param {*} storeAPI The redux store instance
 * @returns the result of calling next(action)
 */
const maintenanceModeMiddleware = storeAPI => next => action => {
    if (action.type === TestController.MaintenanceModeChanged) {
        if (storeAPI.getState().system.maintenanceMode !== action.payload?.maintenanceMode) {
            storeAPI.dispatch({
                type: (action.payload?.maintenanceMode ? TestController.Toast.Warning : TestController.Toast.Success),
                payload: `The system is ${action.payload?.maintenanceMode ? "now" : "no longer"} in maintenance mode`
            });
        }
    }
    // Do something in here, when each action is dispatched
    return next(action)
}

export default maintenanceModeMiddleware;