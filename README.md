# public_proxy
利用公网服务器代理各种局域网的网络协议，例如 远程桌面协议


解释：

公网机器P，私网机器A及A可以访问的私有资源d, 客户机B

B希望通过P和A的帮助访问d

原理    A ----> P (main port)

        B ----> P (from port)

        A ----> P (to port)

        A ----> d (dest port)

        P 交换from port 和 to port之间的数据

        A 交换 p (to port) 和 d(dest port) 之间的数据

利用两个中继程序实现了私有资源的公网访问

