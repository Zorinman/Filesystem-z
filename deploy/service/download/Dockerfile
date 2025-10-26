FROM centos:7

ADD bin/download /

RUN chmod 777 /download

ENV PARAMS=""

ENTRYPOINT ["sh","-c","/download $PARAMS"] 
#在Kubernetes中部署时，如果yaml文件中指定了command和args，ENTRYPOINT中的命令不会被执行，反之则执行