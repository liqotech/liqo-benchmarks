# kubemark

This patch introduces a slight modification to the [kubemark](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-scalability/kubemark-guide.md) hollow kubelet (v1.21.4) to make Pod IPs to be assigned from the CIDR associated with the hosting node, rather than using a fake one.

A kubemark docker image (including the patch), can be generated through:

```bash
git clone https://github.com/kubernetes/kubernetes.git
cd kubernetes

git checkout v1.21.4
git am 0001-hollow-kubelet-respect-pod-CIDR.patch

make WHAT=cmd/kubemark
mv _output/bin/kubemark cluster/images/kubemark/
docker build cluster/images/kubemark/ --tag <kubemark:tag>
docker push <kubemark:tag>
```
