下面是从零开始的完整操作流程，假设你已经把编译好的 `zecode` 二进制传到了服务器上，按顺序执行即可。

## 0. 前置确认

```bash
# 确认二进制在，且有执行权限
ls -lh /opt/zecode/zecode

# 确认配置文件在
ls -lh /opt/zecode/config/config.toml

# 手动跑一下，确认程序本身没问题（跑通后 Ctrl+C 退出）
cd /opt/zecode
./zecode
```

预期看到启动 banner，包含版本、端口、访问地址。如果这一步就失败，先解决程序本身的问题（端口被占、配置格式错、statics 缺失等），再往下走。

## 1. 创建专用系统用户

```bash
sudo useradd \
  --system \
  --no-create-home \
  --shell /usr/sbin/nologin \
  --home-dir /opt/zecode \
  zecode
```

验证：

```bash
id zecode
```

预期输出类似：

```
uid=998(zecode) gid=998(zecode) groups=998(zecode)
```

再确认它不能登录：

```bash
sudo su - zecode
# 预期输出: This account is currently not available.
```

## 2. 设置目录与权限

```bash
# 目录和文件归 zecode
sudo chown -R zecode:zecode /opt/zecode

# 二进制：所有者可执行
sudo chmod 755 /opt/zecode/zecode

# 配置目录：只允许 zecode 读
sudo chmod 750 /opt/zecode/config
sudo chmod 640 /opt/zecode/config/config.toml

# 检查结果
ls -la /opt/zecode
ls -la /opt/zecode/config
```

预期：

```
-rwxr-xr-x 1 zecode zecode ... zecode
drwxr-x--- 2 zecode zecode ... config
-rw-r----- 1 zecode zecode ... config.toml
```

> 如果程序需要写日志/缓存/上传目录，单独建一个：
> ```bash
> sudo mkdir -p /opt/zecode/data
> sudo chown zecode:zecode /opt/zecode/data
> ```

## 3. 编写 systemd 服务文件

```bash
sudo nano /etc/systemd/system/zecode.service
```

写入以下内容：

```ini
[Unit]
Description=ZeCode QR Code Service
Documentation=https://github.com/zmsr/zecode
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=zecode
Group=zecode
WorkingDirectory=/opt/zecode
ExecStart=/opt/zecode/zecode

# 崩溃后自动重启
Restart=always
RestartSec=5

# 通过环境变量指定配置（可选，默认就是 config/config.toml）
# Environment="ZECODE_CONFIG=/opt/zecode/config/config.toml"

# ---- 安全加固 ----
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/zecode/data
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictAddressFamilies=AF_INET AF_INET6
ProtectClock=true
ProtectHostname=true

# ---- 日志 ----
StandardOutput=journal
StandardError=journal
SyslogIdentifier=zecode

[Install]
WantedBy=multi-user.target
```

**注意**：

- `ProtectSystem=strict` 会把整个文件系统挂成只读，只允许 `ReadWritePaths` 里列的目录写。如果你的程序**不需要写任何文件**（`zecode` 是内存生成二维码，确实不需要），可以把 `ReadWritePaths=` 这行删掉，或者保留 `/opt/zecode/data` 作为将来扩展用。
- 如果程序要绑 **小于 1024 的端口**（比如 80），普通用户没权限，需要加：
  ```ini
  AmbientCapabilities=CAP_NET_BIND_SERVICE
  CapabilityBoundingSet=CAP_NET_BIND_SERVICE
  ```
  但 `zecode` 默认用 8080，不需要。

## 4. 重载并启动

```bash
# 让 systemd 识别新服务
sudo systemctl daemon-reload

# 设置开机自启
sudo systemctl enable zecode

# 立即启动
sudo systemctl start zecode
```

`enable` 的预期输出：

```
Created symlink /etc/systemd/system/multi-user.target.wants/zecode.service → /etc/systemd/system/zecode.service.
```

## 5. 查看运行状态

### 快速看状态

```bash
sudo systemctl status zecode
```

正常输出：

```
● zecode.service - ZeCode QR Code Service
     Loaded: loaded (/etc/systemd/system/zecode.service; enabled; preset: enabled)
     Active: active (running) since ...
   Main PID: 12345 (zecode)
      Tasks: 5 (limit: 4523)
     Memory: 12.3M
        CPU: 15ms
     CGroup: /system.slice/zecode.service
             └─12345 /opt/zecode/zecode
```

关键看三个点：

| 字段 | 期望值 |
|---|---|
| `Loaded` | `enabled` — 开机自启已生效 |
| `Active` | `active (running)` — 正在运行 |
| 进程用户 | 应显示为 zecode，可用 `ps` 再确认 |

### 确认进程属主

```bash
ps -eo pid,user,comm | grep zecode
```

预期：

```
12345 zecode   zecode
```

如果是 `root`，说明 `User=` 没生效，检查服务文件。

### 实时看日志

```bash
sudo journalctl -u zecode -f
```

启动时应看到程序的 banner 输出。按 `Ctrl+C` 退出。

### 看历史日志

```bash
# 最近 50 行
sudo journalctl -u zecode -n 50

# 只看今天
sudo journalctl -u zecode --since today

# 只看错误级别
sudo journalctl -u zecode -p err

# 从本次开机以来
sudo journalctl -u zecode -b
```

### 从外部验证服务

```bash
# 本机测试
curl -I http://localhost:8080/

# 或直接测试 API
curl "http://localhost:8080/api/generate?content=hello" -o /tmp/qr.png
file /tmp/qr.png
# 预期: PNG image data, ...
```

## 6. 验证开机自启

最保险的方式是真重启一次：

```bash
sudo reboot
```

等服务器起来后重新 SSH 上去：

```bash
systemctl status zecode
curl -I http://localhost:8080/
```

如果不想重启，用这个模拟：

```bash
# 停掉服务
sudo systemctl stop zecode

# 检查是否真的停了
systemctl status zecode    # 应显示 inactive (dead)

# 再启动，验证能正常拉起
sudo systemctl start zecode
```

## 7. 日常操作速查

```bash
# 启动 / 停止 / 重启
sudo systemctl start   zecode
sudo systemctl stop    zecode
sudo systemctl restart zecode

# 重新加载配置（不中断服务，前提是程序支持 SIGHUP）
sudo systemctl reload  zecode

# 查看状态与日志
sudo systemctl status  zecode
sudo journalctl -u zecode -f

# 开机自启开关
sudo systemctl enable  zecode
sudo systemctl disable zecode

# 检查是否已设置自启
systemctl is-enabled zecode

# 检查是否正在运行
systemctl is-active zecode
```

## 8. 更新程序流程

```bash
# 1. 上传新二进制到临时位置
scp dist/zecode user@server:/tmp/zecode.new

# 2. 服务器上操作
sudo systemctl stop zecode
sudo mv /tmp/zecode.new /opt/zecode/zecode
sudo chown zecode:zecode /opt/zecode/zecode
sudo chmod 755 /opt/zecode/zecode
sudo systemctl start zecode
sudo systemctl status zecode
```

如果配置文件也跟着更新，注意保持属主和权限：

```bash
sudo chown zecode:zecode /opt/zecode/config/config.toml
sudo chmod 640 /opt/zecode/config/config.toml
```

## 9. 卸载流程

```bash
# 停服务、禁自启
sudo systemctl stop zecode
sudo systemctl disable zecode

# 删服务文件
sudo rm /etc/systemd/system/zecode.service
sudo systemctl daemon-reload
sudo systemctl reset-failed

# 删程序目录
sudo rm -rf /opt/zecode

# 删用户（可选，保留也不影响）
sudo userdel zecode
```

## 10. 常见问题排查

| 现象 | 原因 | 处理 |
|---|---|---|
| `Failed to determine user credentials` | `User=zecode` 但用户不存在 | 重新执行第 1 步建用户 |
| `status=203/EXEC` | ExecStart 路径错 / 二进制没执行权限 | `ls -l` 检查路径与 `chmod 755` |
| `status=200/CHDIR` | WorkingDirectory 不存在 | 检查 `/opt/zecode` 是否存在 |
| 启动即退出、反复重启 | 配置文件错 / 端口被占 | `journalctl -u zecode -n 100` 看具体报错 |
| 日志里报 `permission denied` | 文件属主/权限不对 | `chown -R zecode:zecode /opt/zecode` |
| 服务起了但访问不了 | 端口没对 / 防火墙 | `ss -tlnp \| grep zecode` 确认监听，`sudo ufw status` 查防火墙 |
| 想换端口 | 改 `config.toml` 后重启服务 | `sudo systemctl restart zecode` |
| 报 `Address already in use` | 旧进程没退干净或别的程序占了 | `sudo lsof -i :8080` 找到占用者 |

排查时最有用的一条命令：

```bash
sudo journalctl -u zecode -n 100 --no-pager
```

它会把你程序自己打印的启动信息、错误日志全列出来，多数问题一眼能定位。

## 11. 完整流程串一遍（复制即用）

```bash
# ---- 1. 建用户 ----
sudo useradd --system --no-create-home --shell /usr/sbin/nologin --home-dir /opt/zecode zecode

# ---- 2. 权限 ----
sudo chown -R zecode:zecode /opt/zecode
sudo chmod 755 /opt/zecode/zecode
sudo chmod 750 /opt/zecode/config
sudo chmod 640 /opt/zecode/config/config.toml

# ---- 3. 写服务文件 ----
sudo tee /etc/systemd/system/zecode.service > /dev/null <<'EOF'
[Unit]
Description=ZeCode QR Code Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=zecode
Group=zecode
WorkingDirectory=/opt/zecode
ExecStart=/opt/zecode/zecode
Restart=always
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictAddressFamilies=AF_INET AF_INET6
ProtectClock=true
ProtectHostname=true
StandardOutput=journal
StandardError=journal
SyslogIdentifier=zecode

[Install]
WantedBy=multi-user.target
EOF

# ---- 4. 启动 ----
sudo systemctl daemon-reload
sudo systemctl enable zecode
sudo systemctl start zecode

# ---- 5. 验证 ----
sudo systemctl status zecode
ps -eo pid,user,comm | grep zecode
curl -I http://localhost:8080/
sudo journalctl -u zecode -n 30 --no-pager
```

跑完这五步，服务就装好了。之后重启服务器会自动拉起，进程崩溃也会在 5 秒后自动恢复。