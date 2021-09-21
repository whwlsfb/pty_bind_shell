# pty_bind_shell
Pty bind shell for golang

该项目主要用途为在后渗透阶段增加红队工作效率，同时具有一定的免探测的能力（SSH）。

该项目功能为在目标主机**监听**或**反弹**（待实现）一条SSH通道，该SSH服务通道不受系统类型、系统版本、系统SSH服务类型等因素限制，同时具有完整的Pty Shell(仅支持unix类系统)、SFTP文件管理功能。


![Demo](demo.png)
