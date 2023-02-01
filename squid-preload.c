#include <dlfcn.h>
#include <netdb.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <string.h>
#include <assert.h>

// Bypass squid v5 crash bug 
// when connecting to IPv6 addr in IPv4-only environment (Docker container, etc.)
// Ref: https://bugs.squid-cache.org/show_bug.cgi?id=5154

int (*original_getaddrinfo)(const char *restrict node,
                            const char *restrict service,
                            const struct addrinfo *restrict hints,
                            struct addrinfo **restrict res);

int getaddrinfo(const char *restrict node,
                const char *restrict service,
                const struct addrinfo *restrict hints,
                struct addrinfo **restrict res)
{
    if (!original_getaddrinfo)
    {
        original_getaddrinfo = dlsym(RTLD_NEXT, "getaddrinfo");
        assert(original_getaddrinfo);
    }
    struct addrinfo real_hints;
    if (hints)
    {
        memcpy(&real_hints, hints, sizeof(struct addrinfo));
    }
    else
    {
        memset(&real_hints, 0, sizeof(struct addrinfo));
    }
    // IPv4 only, let's pretend that IPv6 never exists.
    real_hints.ai_family = AF_INET;
    return original_getaddrinfo(node, service, &real_hints, res);
}
