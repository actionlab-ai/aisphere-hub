**You:**

https://github.com/casdoor/casdoor 给我来个linux 二进制安装教程

* * *

**ChatGPT:**

下面给你一版 **Linux 二进制安装 Casdoor** 的标准教程。Casdoor 官方文档说明：GitHub Releases 提供 Linux x86_64、arm64 等预编译包，解压后直接运行 `casdoor`，包里已经包含 Web 前端和示例 `conf/app.conf`。Casdoor 默认生产端口是 `8000`，首次部署默认账号一般是 `built-in/admin`，密码 `123`，上线后要立刻改密码。[![](https://www.google.com/s2/favicons?domain=https://casdoor.ai&sz=128)casdoor.ai+1](https://casdoor.ai/docs/basic/server-installation)

## 1. 准备数据库

生产建议用 **MySQL / PostgreSQL**，测试可以用 SQLite。Casdoor 支持 MySQL、MariaDB、PostgreSQL、SQLite、TiDB 等数据库。[![](https://www.google.com/s2/favicons?domain=https://casdoor.ai&sz=128)casdoor.ai](https://casdoor.ai/docs/basic/server-installation)

以 MySQL 为例：

```Bash
mysql -uroot -p

CREATE DATABASE casdoor DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'casdoor'@'%' IDENTIFIED BY 'Casdoor@123456';
GRANT ALL PRIVILEGES ON casdoor.* TO 'casdoor'@'%';
FLUSH PRIVILEGES;
EXIT;
```

如果你的 MySQL 和 Casdoor 在同一台机器，可以把 `'%'` 改成 `'localhost'`。

* * *

## 2. 下载 Casdoor 二进制包

Linux x86_64：

```Bash
cd /opt

VER="v3.93.0"
wget -O casdoor_Linux_x86_64.tar.gz \
  "https://github.com/casdoor/casdoor/releases/download/${VER}/casdoor_Linux_x86_64.tar.gz"

tar -xzf casdoor_Linux_x86_64.tar.gz
mv casdoor_Linux_x86_64 casdoor
```

ARM64 机器，比如鲲鹏、飞腾、部分 ARM 服务器：

```Bash
cd /opt

VER="v3.93.0"
wget -O casdoor_Linux_arm64.tar.gz \
  "https://github.com/casdoor/casdoor/releases/download/${VER}/casdoor_Linux_arm64.tar.gz"

tar -xzf casdoor_Linux_arm64.tar.gz
mv casdoor_Linux_arm64 casdoor
```

确认文件：

```Bash
cd /opt/casdoor
ls -lah
ls -lah conf/
```

应该能看到：

```Bash
casdoor
conf/app.conf
```

* * *

## 3. 修改配置文件

编辑：

```Bash
vim /opt/casdoor/conf/app.conf
```

关键配置改成这样：

```INI
appname = casdoor
httpport = 8000
runmode = prod

driverName = mysql
dataSourceName = casdoor:Casdoor@123456@tcp(127.0.0.1:3306)/
dbName = casdoor

origin = http://你的服务器IP:8000
```

如果你准备用域名和 Nginx 反代，比如：

```
https://casdoor.example.com
```

那就改成：

```INI
origin = https://casdoor.example.com
```

`httpport` 是后端监听端口，默认是 `8000`；`driverName`、`dataSourceName`、`dbName` 是最小数据库配置；`origin` 是 Casdoor 对外访问地址，做 OAuth/OIDC 回调时很关键。[![](https://www.google.com/s2/favicons?domain=https://casdoor.ai&sz=128)casdoor.ai](https://casdoor.ai/docs/basic/configuration)

* * *

## 4. 手动启动测试

```Bash
cd /opt/casdoor
chmod +x ./casdoor
./casdoor
```

另开一个终端测试：

```Bash
curl -I http://127.0.0.1:8000
```

浏览器访问：

```
http://服务器IP:8000
```

默认登录：

```
组织：built-in
用户：admin
密码：123
```

也就是登录名一般填：

```
built-in/admin
```

登录后第一件事：**修改 admin 密码**。

* * *

## 5. 配置 systemd 开机自启

创建运行用户：

```Bash
useradd -r -s /sbin/nologin -d /opt/casdoor casdoor || true
chown -R casdoor:casdoor /opt/casdoor
```

创建 systemd 文件：

```Bash
cat > /etc/systemd/system/casdoor.service <<'EOF'
[Unit]
Description=Casdoor IAM Server
After=network.target mysql.service
Wants=network.target

[Service]
Type=simple
User=casdoor
Group=casdoor
WorkingDirectory=/opt/casdoor
ExecStart=/opt/casdoor/casdoor --config /opt/casdoor/conf/app.conf
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF
```

启动：

```Bash
systemctl daemon-reload
systemctl enable --now casdoor
systemctl status casdoor
```

看日志：

```Bash
journalctl -u casdoor -f
```

* * *

## 6. 可选：Nginx 反向代理

官方建议生产环境可以把 Nginx 放在 Casdoor 前面，把域名流量转发到后端 `8000` 端口。[![](https://www.google.com/s2/favicons?domain=https://casdoor.ai&sz=128)casdoor.ai](https://casdoor.ai/docs/deployment/nginx)

安装 Nginx：

```Bash
yum install -y nginx
# 或 Ubuntu/Debian:
# apt update && apt install -y nginx
```

配置：

```Bash
cat > /etc/nginx/conf.d/casdoor.conf <<'EOF'
server {
    listen 80;
    server_name casdoor.example.com;

    location / {
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_redirect off;
        proxy_pass http://127.0.0.1:8000;
    }
}
EOF
```

检查并重载：

```Bash
nginx -t
systemctl enable --now nginx
systemctl reload nginx
```

然后把 `/opt/casdoor/conf/app.conf` 里的：

```INI
origin = http://你的服务器IP:8000
```

改成：

```INI
origin = http://casdoor.example.com
```

重启 Casdoor：

```Bash
systemctl restart casdoor
```

* * *

## 7. 防火墙放行

直接访问 8000：

```Bash
firewall-cmd --permanent --add-port=8000/tcp
firewall-cmd --reload
```

如果走 Nginx，只放 80/443：

```Bash
firewall-cmd --permanent --add-service=http
firewall-cmd --permanent --add-service=https
firewall-cmd --reload
```

* * *

## 8. 常用运维命令

```Bash
# 查看状态
systemctl status casdoor

# 重启
systemctl restart casdoor

# 停止
systemctl stop casdoor

# 实时日志
journalctl -u casdoor -f

# 查看监听
ss -lntp | grep 8000
```

* * *

## 9. 升级方式

```Bash
cd /opt

systemctl stop casdoor

cp -a /opt/casdoor /opt/casdoor.bak.$(date +%F-%H%M)

VER="v3.93.0"
wget -O casdoor_Linux_x86_64.tar.gz \
  "https://github.com/casdoor/casdoor/releases/download/${VER}/casdoor_Linux_x86_64.tar.gz"

tar -xzf casdoor_Linux_x86_64.tar.gz

cp -a /opt/casdoor/conf/app.conf /tmp/app.conf.casdoor
rm -rf /opt/casdoor
mv casdoor_Linux_x86_64 /opt/casdoor
cp -a /tmp/app.conf.casdoor /opt/casdoor/conf/app.conf

chown -R casdoor:casdoor /opt/casdoor
systemctl start casdoor
systemctl status casdoor
```

* * *

## 10. 最小可用配置总结

你现在最少需要改这些：

```INI
httpport = 8000
runmode = prod

driverName = mysql
dataSourceName = casdoor:Casdoor@123456@tcp(127.0.0.1:3306)/
dbName = casdoor

origin = http://你的服务器IP:8000
```

部署好后访问：

```
http://你的服务器IP:8000
```

登录：

```
built-in/admin
123
```

然后马上改密码。