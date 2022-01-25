import {TestController} from '../actions';
import client from '../apiclient';

export const reducer = (state = {commits:[]}, action) => {
    switch(action.type) {
        case TestController.InitialStateLoaded:
            return {commits:action.payload?.commits || []};
        default:
            return state;
    }
}