import {TestController} from '../actions';
import { createSelector } from 'reselect'
import client from '../apiclient';

export const reducer = (state = {users:[],me:{}}, action) => {
    switch(action.type) {
        case TestController.InitialStateLoaded:
            return {users:action.payload?.users || [], me:action.payload?.me || {}};
        case TestController.UsersUpdated:
            return {
                ...state,
                users:(action.payload || [])
            }
        default:
            return state;
    }
}

export const deleteUser = thumb => async (dispatch) => {
    try {
        let result = await client.del(`users/${thumb}`);
        if(result.ok){
            result = await(client.get("users"))
            dispatch({type:TestController.UsersUpdated, payload: result});
        } else {
            dispatch({type:TestController.Toast.Error, payload: "Server did not return OK"});
        }
    } catch (e) {
        dispatch({type:TestController.Toast.Error, payload: e});
    }
}

export const addUser = cert => async (dispatch) => {
    try {
        var result = await client.post("users", cert, true)
        if(result.ok){
           result = await(client.get("users"));
           dispatch({type:TestController.UsersUpdated, payload: result});
        } else {
           dispatch({type:TestController.Toast.Error, payload: "Server did not return OK"});
        }
    } catch (e) {
        dispatch({type:TestController.Toast.Error, payload: e});
    }
}