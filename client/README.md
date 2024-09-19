## 类型

- RESTClient： 最基础的客户端，提供最基本的封装
- ClientSet :是一个Client的集合，在ClientSet中包含了所有K8S内置资源的Client，通过ClientSet便可以很方便的操作如Pod、Service这些资源
- dynamicClient：动态客户端，可以操作任意K8S的资源，包括CRD定义的资源
- DiscoveryClient：用于发现K8S提供的资源组、资源版本和资源信息，比如:kubectl api-resources









