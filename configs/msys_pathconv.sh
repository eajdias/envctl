# Disable path conversion for container paths and common Docker/Kubernetes flags
export MSYS2_ARG_CONV_EXCL="/bin;/usr;/var;/etc;/app;/tmp;/opt;--entrypoint;-v;--volume;--mount;--workdir;-w"
