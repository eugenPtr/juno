"use strict";(self.webpackChunkmy_website=self.webpackChunkmy_website||[]).push([[289],{3905:(e,n,t)=>{t.d(n,{Zo:()=>c,kt:()=>m});var r=t(7294);function o(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function a(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);n&&(r=r.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,r)}return t}function i(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?a(Object(t),!0).forEach((function(n){o(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):a(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function s(e,n){if(null==e)return{};var t,r,o=function(e,n){if(null==e)return{};var t,r,o={},a=Object.keys(e);for(r=0;r<a.length;r++)t=a[r],n.indexOf(t)>=0||(o[t]=e[t]);return o}(e,n);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);for(r=0;r<a.length;r++)t=a[r],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(o[t]=e[t])}return o}var l=r.createContext({}),p=function(e){var n=r.useContext(l),t=n;return e&&(t="function"==typeof e?e(n):i(i({},n),e)),t},c=function(e){var n=p(e.components);return r.createElement(l.Provider,{value:n},e.children)},u="mdxType",f={inlineCode:"code",wrapper:function(e){var n=e.children;return r.createElement(r.Fragment,{},n)}},h=r.forwardRef((function(e,n){var t=e.components,o=e.mdxType,a=e.originalType,l=e.parentName,c=s(e,["components","mdxType","originalType","parentName"]),u=p(t),h=o,m=u["".concat(l,".").concat(h)]||u[h]||f[h]||a;return t?r.createElement(m,i(i({ref:n},c),{},{components:t})):r.createElement(m,i({ref:n},c))}));function m(e,n){var t=arguments,o=n&&n.mdxType;if("string"==typeof e||o){var a=t.length,i=new Array(a);i[0]=h;var s={};for(var l in n)hasOwnProperty.call(n,l)&&(s[l]=n[l]);s.originalType=e,s[u]="string"==typeof e?e:o,i[1]=s;for(var p=2;p<a;p++)i[p]=t[p];return r.createElement.apply(null,i)}return r.createElement.apply(null,t)}h.displayName="MDXCreateElement"},8527:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>l,contentTitle:()=>i,default:()=>f,frontMatter:()=>a,metadata:()=>s,toc:()=>p});var r=t(7462),o=(t(7294),t(3905));const a={slug:"/config",sidebar_position:3,title:"Example Configuration"},i=void 0,s={unversionedId:"example_config",id:"version-0.9.2/example_config",title:"Example Configuration",description:"The Juno binary uses reasonable defaults and can be used without configuration.",source:"@site/versioned_docs/version-0.9.2/example_config.md",sourceDirName:".",slug:"/config",permalink:"/0.9.2/config",draft:!1,tags:[],version:"0.9.2",sidebarPosition:3,frontMatter:{slug:"/config",sidebar_position:3,title:"Example Configuration"},sidebar:"tutorialSidebar",previous:{title:"Quick Start",permalink:"/0.9.2/"},next:{title:"Database Snapshots",permalink:"/0.9.2/snapshots"}},l={},p=[],c={toc:p},u="wrapper";function f(e){let{components:n,...t}=e;return(0,o.kt)(u,(0,r.Z)({},c,t,{components:n,mdxType:"MDXLayout"}),(0,o.kt)("p",null,"The Juno binary uses reasonable defaults and can be used without configuration.\nFor basic fine-tuning, the ",(0,o.kt)("inlineCode",{parentName:"p"},"--db-path")," and ",(0,o.kt)("inlineCode",{parentName:"p"},"--http-port")," options are usually sufficient."),(0,o.kt)("p",null,"All available options are in the YAML file below with their default values.\nProvide the config using the ",(0,o.kt)("inlineCode",{parentName:"p"},"--config <filename>")," option (Juno looks in ",(0,o.kt)("inlineCode",{parentName:"p"},"$XDG_CONFIG_HOME")," by default)."),(0,o.kt)("p",null,"Juno can also be configured using command line params by prepending ",(0,o.kt)("inlineCode",{parentName:"p"},"--")," to the option name (e.g., ",(0,o.kt)("inlineCode",{parentName:"p"},"--log-level info"),").\nCommand line params override values in the configuration file. "),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-yaml"},'# The yaml configuration file\nconfig: ""\n\n# Options: debug, info, warn, error\nlog-level: info\n\n# Enables the HTTP RPC server on the default port and interface\nhttp: false\n\n# The interface on which the HTTP RPC server will listen for requests\nhttp-host: localhost\n\n# The port on which the HTTP server will listen for requests\nhttp-port: 6060\n\n# Enables the Websocket RPC server on the default port\nws: false\n\n# The interface on which the Websocket RPC server will listen for requests\nws-host: localhost\n\n# The port on which the websocket server will listen for requests\nws-port: 6061\n\n# Location of the database files\ndb-path: /home/<user>/.local/share/juno\n\n# Options: mainnet, goerli, goerli2, integration, sepolia, sepolia-integration\nnetwork: mainnet\n\n# Websocket endpoint of the Ethereum node\neth-node: ""\n\n# Enables the pprof endpoint on the default port\npprof: false\n\n# The interface on which the pprof HTTP server will listen for requests\npprof-host: localhost\n\n# The port on which the pprof HTTP server will listen for requests\npprof-port: 6062\n\n# Uses --colour=false command to disable colourized outputs (ANSI Escape Codes)\ncolour: true\n\n# Sets how frequently pending block will be updated (disabled by default)\npending-poll-interval: 0s\n\n# Enable p2p server\np2p: false\n\n# Specify p2p source address as multiaddr\np2p-addr: ""\n\n# Specify list of p2p boot peers splitted by a comma\np2p-boot-peers: ""\n\n# Enables the prometheus metrics endpoint on the default port\nmetrics: false\n\n# The interface on which the prometheus endpoint will listen for requests\nmetrics-host: localhost\n\n# The port on which the prometheus endpoint will listen for requests\nmetrics-port: 9090\n\n# Enable the HTTP GRPC server on the default port\ngrpc: false\n\n# The interface on which the GRPC server will listen for requests\ngrpc-host: localhost\n\n# The port on which the GRPC server will listen for requests\ngrpc-port: 6064\n\n# Maximum number of VM instances for concurrent RPC calls.\n# Default is set to three times the number of CPU cores.\nmax-vms: 48\n\n# Maximum number of requests to queue for RPC calls after reaching max-vms.\n# Default is set to double the value of max-vms.\nmax-vm-queue: 96\n\n# gRPC URL of a remote Juno node\nremote-db: ""\n\n# Maximum number of blocks scanned in single starknet_getEvents call\nrpc-max-block-scan: 18446744073709551615\n\n# Determines the amount of memory (in megabytes) allocated for caching data in the database\ndb-cache-size: 8\n\n# A soft limit on the number of open files that can be used by the DB\ndb-max-handles: 1024\n\n# API key for gateway/feeder to avoid throttling\ngw-api-key: ""\n')))}f.isMDXComponent=!0}}]);