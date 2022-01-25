import { TestController } from '../actions';
import { toast } from 'react-toastify';

/**
 * toastMiddleware is React Redux middleware responsible 
 * for translating Redux events to displayed toast messages
 * in the UI
 * @param {*} storeAPI The redux store instance
 * @returns the result of calling next(action)
 */
const toastMiddleware = storeAPI => next => action => {
    switch (action.type) {
        case TestController.Toast.Success:
            toast.success(action.payload)
            break;
        case TestController.Toast.Error:
            toast.error(action.payload)
            break;
        case TestController.Toast.Warning:
            toast.warn(action.payload)
            break;
        case TestController.Toast.Info:
            toast.error(action.payload)
            break;
    }
    // Do something in here, when each action is dispatched
    return next(action)
}

export default toastMiddleware;