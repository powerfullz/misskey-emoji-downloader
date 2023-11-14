# Misskey emoji downloader

现学现卖写的一个 Misskey 偷表情工具，inspired by [Starainrt/emojidownloader](https://github.com/Starainrt/emojidownloader)

用法：

```shell
python emoji.py
```

只有一个依赖：`requests`

如果需要使用代理，在终端内设置环境变量即可，以 Linux 为例：

```shell
export all_proxy=socks5://127.0.0.1:7890
```

写得像坨粑粑，但能用就行（狗头），以后看情况完善一下吧。

## To do List

- [ ] 代理
- [ ] 改用`async`
- [ ] Go 语言重构
