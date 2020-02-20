package nsenter

/*
extern void nsexec();
__attribute__((constructor)) void enter_ns(void) {
	nsexec();
}
 */
import "C"