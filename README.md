# 开发
```
git add .
git commit -m "feat: xxx"
git push origin main
→ 触发 ci.yml，跑 lint / build / test / cross-build
```

# 发版
```
git tag v1.0.0
git push origin v1.0.0
→ 触发 release.yml，构建 6 个平台包，自动创建 GitHub Release 并附上 checksums.txt
```