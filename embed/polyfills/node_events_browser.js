/* esm.sh - esbuild bundle(events@3.3.0) es2015 production */
var N=Object.create,p=Object.defineProperty,M=Object.getPrototypeOf,A=Object.prototype.hasOwnProperty,P=Object.getOwnPropertyNames,T=Object.getOwnPropertyDescriptor;var F=t=>p(t,"__esModule",{value:!0});var I=(t,e)=>()=>(e||t((e={exports:{}}).exports,e),e.exports);var K=(t,e,n)=>{if(e&&typeof e=="object"||typeof e=="function")for(let r of P(e))!A.call(t,r)&&r!=="default"&&p(t,r,{get:()=>e[r],enumerable:!(n=T(e,r))||n.enumerable});return t},L=t=>K(F(p(t!=null?N(M(t)):{},"default",t&&t.__esModule&&"default"in t?{get:()=>t.default,enumerable:!0}:{value:t,enumerable:!0})),t);var h=I((G,d)=>{"use strict";var a=typeof Reflect=="object"?Reflect:null,y=a&&typeof a.apply=="function"?a.apply:function(e,n,r){return Function.prototype.apply.call(e,n,r)},l;a&&typeof a.ownKeys=="function"?l=a.ownKeys:Object.getOwnPropertySymbols?l=function(e){return Object.getOwnPropertyNames(e).concat(Object.getOwnPropertySymbols(e))}:l=function(e){return Object.getOwnPropertyNames(e)};function W(t){console&&console.warn&&console.warn(t)}var g=Number.isNaN||function(e){return e!==e};function o(){o.init.call(this)}d.exports=o;d.exports.once=S;o.EventEmitter=o;o.prototype._events=void 0;o.prototype._eventsCount=0;o.prototype._maxListeners=void 0;var _=10;function v(t){if(typeof t!="function")throw new TypeError('The "listener" argument must be of type Function. Received type '+typeof t)}Object.defineProperty(o,"defaultMaxListeners",{enumerable:!0,get:function(){return _},set:function(t){if(typeof t!="number"||t<0||g(t))throw new RangeError('The value of "defaultMaxListeners" is out of range. It must be a non-negative number. Received '+t+".");_=t}});o.init=function(){(this._events===void 0||this._events===Object.getPrototypeOf(this)._events)&&(this._events=Object.create(null),this._eventsCount=0),this._maxListeners=this._maxListeners||void 0};o.prototype.setMaxListeners=function(e){if(typeof e!="number"||e<0||g(e))throw new RangeError('The value of "n" is out of range. It must be a non-negative number. Received '+e+".");return this._maxListeners=e,this};function w(t){return t._maxListeners===void 0?o.defaultMaxListeners:t._maxListeners}o.prototype.getMaxListeners=function(){return w(this)};o.prototype.emit=function(e){for(var n=[],r=1;r<arguments.length;r++)n.push(arguments[r]);var i=e==="error",f=this._events;if(f!==void 0)i=i&&f.error===void 0;else if(!i)return!1;if(i){var s;if(n.length>0&&(s=n[0]),s instanceof Error)throw s;var u=new Error("Unhandled error."+(s?" ("+s.message+")":""));throw u.context=s,u}var c=f[e];if(c===void 0)return!1;if(typeof c=="function")y(c,this,n);else for(var m=c.length,R=b(c,m),r=0;r<m;++r)y(R[r],this,n);return!0};function E(t,e,n,r){var i,f,s;if(v(n),f=t._events,f===void 0?(f=t._events=Object.create(null),t._eventsCount=0):(f.newListener!==void 0&&(t.emit("newListener",e,n.listener?n.listener:n),f=t._events),s=f[e]),s===void 0)s=f[e]=n,++t._eventsCount;else if(typeof s=="function"?s=f[e]=r?[n,s]:[s,n]:r?s.unshift(n):s.push(n),i=w(t),i>0&&s.length>i&&!s.warned){s.warned=!0;var u=new Error("Possible EventEmitter memory leak detected. "+s.length+" "+String(e)+" listeners added. Use emitter.setMaxListeners() to increase limit");u.name="MaxListenersExceededWarning",u.emitter=t,u.type=e,u.count=s.length,W(u)}return t}o.prototype.addListener=function(e,n){return E(this,e,n,!1)};o.prototype.on=o.prototype.addListener;o.prototype.prependListener=function(e,n){return E(this,e,n,!0)};function U(){if(!this.fired)return this.target.removeListener(this.type,this.wrapFn),this.fired=!0,arguments.length===0?this.listener.call(this.target):this.listener.apply(this.target,arguments)}function O(t,e,n){var r={fired:!1,wrapFn:void 0,target:t,type:e,listener:n},i=U.bind(r);return i.listener=n,r.wrapFn=i,i}o.prototype.once=function(e,n){return v(n),this.on(e,O(this,e,n)),this};o.prototype.prependOnceListener=function(e,n){return v(n),this.prependListener(e,O(this,e,n)),this};o.prototype.removeListener=function(e,n){var r,i,f,s,u;if(v(n),i=this._events,i===void 0)return this;if(r=i[e],r===void 0)return this;if(r===n||r.listener===n)--this._eventsCount==0?this._events=Object.create(null):(delete i[e],i.removeListener&&this.emit("removeListener",e,r.listener||n));else if(typeof r!="function"){for(f=-1,s=r.length-1;s>=0;s--)if(r[s]===n||r[s].listener===n){u=r[s].listener,f=s;break}if(f<0)return this;f===0?r.shift():k(r,f),r.length===1&&(i[e]=r[0]),i.removeListener!==void 0&&this.emit("removeListener",e,u||n)}return this};o.prototype.off=o.prototype.removeListener;o.prototype.removeAllListeners=function(e){var n,r,i;if(r=this._events,r===void 0)return this;if(r.removeListener===void 0)return arguments.length===0?(this._events=Object.create(null),this._eventsCount=0):r[e]!==void 0&&(--this._eventsCount==0?this._events=Object.create(null):delete r[e]),this;if(arguments.length===0){var f=Object.keys(r),s;for(i=0;i<f.length;++i)s=f[i],s!=="removeListener"&&this.removeAllListeners(s);return this.removeAllListeners("removeListener"),this._events=Object.create(null),this._eventsCount=0,this}if(n=r[e],typeof n=="function")this.removeListener(e,n);else if(n!==void 0)for(i=n.length-1;i>=0;i--)this.removeListener(e,n[i]);return this};function x(t,e,n){var r=t._events;if(r===void 0)return[];var i=r[e];return i===void 0?[]:typeof i=="function"?n?[i.listener||i]:[i]:n?H(i):b(i,i.length)}o.prototype.listeners=function(e){return x(this,e,!0)};o.prototype.rawListeners=function(e){return x(this,e,!1)};o.listenerCount=function(t,e){return typeof t.listenerCount=="function"?t.listenerCount(e):C.call(t,e)};o.prototype.listenerCount=C;function C(t){var e=this._events;if(e!==void 0){var n=e[t];if(typeof n=="function")return 1;if(n!==void 0)return n.length}return 0}o.prototype.eventNames=function(){return this._eventsCount>0?l(this._events):[]};function b(t,e){for(var n=new Array(e),r=0;r<e;++r)n[r]=t[r];return n}function k(t,e){for(;e+1<t.length;e++)t[e]=t[e+1];t.pop()}function H(t){for(var e=new Array(t.length),n=0;n<e.length;++n)e[n]=t[n].listener||t[n];return e}function S(t,e){return new Promise(function(n,r){function i(s){t.removeListener(e,f),r(s)}function f(){typeof t.removeListener=="function"&&t.removeListener("error",i),n([].slice.call(arguments))}j(t,e,f,{once:!0}),e!=="error"&&q(t,i,{once:!0})})}function q(t,e,n){typeof t.on=="function"&&j(t,"error",e,n)}function j(t,e,n,r){if(typeof t.on=="function")r.once?t.once(e,n):t.on(e,n);else if(typeof t.addEventListener=="function")t.addEventListener(e,function i(f){r.once&&t.removeEventListener(e,i),n(f)});else throw new TypeError('The "emitter" argument must be of type EventEmitter. Received type '+typeof t)}});var z=L(h()),B=L(h()),{once:Q}=z;var export_default=B.default;export{export_default as default,Q as once};