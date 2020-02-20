#define _GNU_SOURCE
#include <fcntl.h> // for open
#include <unistd.h> // for close
#include <sched.h> // for setns
#include <stdlib.h>
#include <stdio.h>
#include <errno.h>
#include <string.h>

void nsexec(void) {
    char *lumper_pid;
    // 从环境变量中获取要进入的 PID
    lumper_pid = getenv("lumper_pid");
    if(lumper_pid) {
        //fprintf(stdout, "lumper_pid is %s\n", lumper_pid);
    } else {
        fprintf(stdout, "missing lumper_pid");
        return;
    }
    // 从环境变量中获取要执行的命令
    char *lumper_cmd;
    lumper_cmd = getenv("lumper_cmd");
    if(lumper_cmd) {
        //fprintf(stdout, "lumper_cmd is %s\n", lumper_cmd);
    } else {
        fprintf(stdout, "missing lumper_cmd");
        return;
    }
    char nspath[1024];
    char *namespaces[] = {"ipc", "uts", "net", "pid", "mnt"};
    for(int i=0; i<5; i++) {
        sprintf(nspath, "/proc/%s/ns/%s", lumper_pid, namespaces[i]);
        int fd = open(nspath, O_RDONLY);
        if(setns(fd, 0) == -1) {
            fprintf(stderr, "setns on %s namespace failed: %s\n", namespaces[i], strerror(errno));
        } else {
            //fprintf(stdout, "setns on %s namespace succeeded\n", namespaces[i]);
        }
        close(fd);
    }
    // 进入 Namespace 中执行命令
    int res = system(lumper_cmd);
    exit(0);
    return;
}