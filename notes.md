【Gin】
ShouldBindJSON(&req)    绑定 JSON 请求体
ShouldBind(&req)        绑定 form/multipart 请求
c.FormFile("file")      拿上传的文件头
f.Open() / f.Close()    打开文件
c.JSON(code, data)      返回 JSON
jwt.GetAccountID(c)     从上下文拿当前用户 ID

【GORM】
db.WithContext(ctx).Create(&x)           插入
db.First(&x, id)                         按主键查
db.Where("col = ?", v).Find(&list)       条件查列表
db.Model(&X{}).Where(...).Update("col", v)  更新
db.Delete(&X{}, id)                      删除
db.Model(&X{}).Count(&n)                 计数

【文件 / 哈希】
os.MkdirAll(path, 0o755)     建目录（递归）
os.Create(path) / os.Open    建/开文件
io.Copy(dst, src)            流式拷贝
md5.New() + io.Copy(h, f)    算 MD5
filepath.Join(a, b)          拼路径

【Redis 封装（项目自己封的）】
cache.GetBytes(ctx, key)    读
cache.SetBytes(ctx, key, b, ttl)  写
cache.Del(ctx, key)         删
cache.Key("fmt:%s", id)     拼 key

【字符串】
fmt.Sprintf / strconv.Itoa / strings.TrimSpace
