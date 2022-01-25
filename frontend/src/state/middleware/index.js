import maintenanceModeMiddleware from './maintenancemode';
import toastMiddleware from './toast';
import websocketMiddleware from './websocket';
import initialStateMiddleware from './initialstate';

export default [
    maintenanceModeMiddleware,
    toastMiddleware,
    websocketMiddleware,
    initialStateMiddleware
];