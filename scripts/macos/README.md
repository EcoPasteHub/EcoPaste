swift 打包：

```bash
# arm
xcrun swift build -c release --arch arm64

# x86
xcrun swift build -c release --arch x86_64
```

object-c 打包：

> 说明：打包一个进行更名，再次打包另一个

```bash
#arm
xcodebuild -arch arm64

# x86
xcodebuild -arch x86_64
```
