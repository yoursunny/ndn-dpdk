#include "../core/common.h"
#include "../core/logger.h"
#include <rte_bpf.h>
#include <ubpf.h>

N_LOG_INIT(StrategyCodeLoad);

struct rte_bpf
{
  struct rte_bpf_prm prm;
  struct rte_bpf_jit jit;
  size_t sz;
  uint32_t stack_sz;
};

static int
StrategyCode_LoadUbpf(struct rte_bpf* bpf, const struct rte_bpf_prm* prm, struct ubpf_vm* vm)
{
  for (uint32_t i = 0; i < prm->nb_xsym; ++i) {
    if (prm->xsym[i].type != RTE_BPF_XTYPE_FUNC) {
      N_LOGE("unsupported xsym type index=%" PRIu32 " type=%d" N_LOG_ERROR_BLANK, i,
             prm->xsym[i].type);
      return ENOTSUP;
    }

    int res = ubpf_register(vm, i, prm->xsym[i].name, prm->xsym[i].func.val);
    if (res != 0) {
      N_LOGE("ubpf_register error index=%" PRIu32 " name=%s" N_LOG_ERROR_BLANK, i,
             prm->xsym[i].name);
      return ENOSYS;
    }
  }

  char* err = NULL;
  int res = ubpf_load(vm, prm->ins, prm->nb_ins * sizeof(prm->ins[0]), &err);
  if (res != 0) {
    N_LOGE("ubpf_load" N_LOG_ERROR_STR, err);
    return EINVAL;
  }

  ubpf_jit_fn jit = ubpf_compile(vm, &err);
  if (jit == NULL) {
    N_LOGE("ubpf_compile error" N_LOG_ERROR_STR, err);
    return ENOEXEC;
  }

  bpf->jit.func = (void*)jit;
  return 0;
}

struct rte_bpf*
rte_bpf_load(const struct rte_bpf_prm* prm)
{
  struct rte_bpf* bpf = (struct rte_bpf*)rte_zmalloc("rte_bpf", sizeof(struct rte_bpf), 0);
  if (bpf == NULL) {
    rte_errno = ENOMEM;
    return NULL;
  }

  struct ubpf_vm* vm = ubpf_create();
  if (vm == NULL) {
    N_LOGE("ubpf_create error" N_LOG_ERROR_BLANK);
    rte_errno = ENOMEM;
    rte_free(bpf);
    return NULL;
  }

  N_LOGD("rte_bpf_load-monkeypatch prm=%p bpf=%p vm=%p", prm, bpf, vm);

  rte_errno = StrategyCode_LoadUbpf(bpf, prm, vm);
  if (rte_errno != 0) {
    ubpf_destroy(vm);
    rte_free(bpf);
    return NULL;
  }

  bpf->prm.xsym = (void*)vm;
  return bpf;
}

void
rte_bpf_destroy(struct rte_bpf* bpf)
{
  struct ubpf_vm* vm = (struct ubpf_vm*)(bpf->prm.xsym);
  N_LOGD("rte_bpf_destroy-monkeypatch bpf=%p vm=%p", bpf, vm);

  ubpf_destroy(vm);
  rte_free(bpf);
}
