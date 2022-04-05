const apiUrl = (window.location.host === "localhost:3000") ? "https://localhost:8443/api/" : `${window.location.protocol}//${window.location.host}/api/`;

const process = (req) => req.json();
const myFetch = async (url, opts) => await fetch(url, Object.assign({}, {credentials:'include'}, opts));
const call = async (url, method, body, raw, rawResult) => {
  let opts = {method};
  if(body) {
    opts.headers = {"Content-Type" : (raw ? "binary/octet-stream" : "application/json")};
    opts.body = raw ? body : JSON.stringify(body);
  }
  const res = await myFetch(`${apiUrl}${url}`, opts);
  return (rawResult ? res : process(res));
}
const get = async(url) => await call(url, 'GET');
const put = async (url, body, raw, rawResult) => await call(url, 'PUT', body, raw, rawResult);
const post = async (url, body, raw, rawResult) => await call(url, 'POST', body, raw, rawResult);
const del = async (url, body, raw, rawResult) => await call(url, 'DELETE', body, raw, rawResult);
const patch = async (url, body, raw, rawResult) => await call(url, 'PATCH', body, raw, rawResult);

export default {apiUrl, call, get, put, post, del, patch};