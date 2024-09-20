这个是应一个简单的小demo,创建service metadata  中 annotations 带有下面标签,会自动去创建对应的ingress
当 **ingress/http: "true"**   被删除后 对应的ingress也会自动删除 . 同样在svc 中 增加该标签也会创建对应的ingress
```shell
metadata：
  annotations:
    ingress/http: "true"
```
示例
```yaml

apiVersion: v1
kind: Service
metadata:
  labels:
    run: test
  annotations:
    ingress/http: "true"
    ingress/domain: "www.test.com"

  name: test
spec:
  ports:
  - port: 80
    protocol: TCP
    targetPort: 80
  selector:
    run: test
status:
  loadBalancer: {}

```

其他配置默认值(根据自己需求修改)
```shell
metadata：
  annotations:
    ingress/http: "true"
    ingress/domain: "www.example.com"
    ingress/Path: "/"
    ingress/Port: "80"
    ingress/targetPath: "/"
```
部署
```shell
kubectl apply  -f https://raw.githubusercontent.com/asws666dsx/client-go-study/refs/heads/main/demo_1/deployment.yaml
```
