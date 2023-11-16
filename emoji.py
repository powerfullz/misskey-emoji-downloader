# -*- coding: utf-8 -*-
import json
import os
import requests
import concurrent.futures

# 下载 Json
userInput = input("输入实例地址（例如：misskey.io）：")
jsonUrl = f"https://{userInput}/api/emojis"
data = json.loads(requests.get(jsonUrl).content)

# 或者直接用本地文件？
# path = r""
# file = open(path, encoding="utf8")
# data = json.load(file)

# categories 去重的笨办法，管他的，先写出来再说
categories = []
for emj in data['emojis']:
    exists = False
    catg = emj['category']
    if catg == None:
        catg = "未分类"
    for catgs in categories:
        if catgs == catg:
            exists = True
            break
    if not(exists):
        categories.append(catg)

# 选择下载分类
print("请选择要下载的分类：")
num = 1
for catgs in categories:
    print(num,".",catgs)
    num = num + 1

userInput = input("输入要下载的分类编号，多个用英文逗号分隔，全部则直接回车：")

finalcatg = []  # 最终想要的分类
if userInput == "":
    finalcatg = categories
else:
    for numbers in userInput.split(","):
        finalcatg.append(categories[int(numbers) - 1])

# 建立分类目录
userInputDir = input("请选择要下载到的目录（默认./myEmojis)：")
directory = "./myEmojis"
if userInputDir != "":
    directory = userInputDir

if not(os.path.exists(directory)):
    os.makedirs(directory)
os.chdir(directory)

for catgs in finalcatg:
    if not(os.path.exists(catgs)):
        os.makedirs(catgs)

# 开始下载
def downloadImage(url, savePath):
    response = requests.get(url, timeout=10)
    if response.status_code == 200:
        with open(savePath, 'wb') as file:
            file.write(response.content)
            print(f"下载成功: {savePath}")
    else:
        print(f"下载失败: {url}")

executor = concurrent.futures.ThreadPoolExecutor(max_workers=16)

for emjs in data['emojis']:
    catg = emjs['category']

    for catgs in finalcatg:
        if catgs == catg:
            url = emjs['url']
            ext = url.split('.')[-1]
            name = emjs['name'] + '.' + ext
            path = f"./{catg}/{name}"
            executor.submit(downloadImage, url, path)
