#include "data.h"
#include "packet.h"

NdnError
PData_FromPacket(PData* data, struct rte_mbuf* pkt, struct rte_mempool* nameMp)
{
  TlvDecodePos d0;
  MbufLoc_Init(&d0, pkt);
  TlvElement dataEle;
  NdnError e = DecodeTlvElementExpectType(&d0, TT_Data, &dataEle);
  RETURN_IF_ERROR;
  data->size = dataEle.size;

  TlvDecodePos d1;
  TlvElement_MakeValueDecoder(&dataEle, &d1);

  TlvElement nameEle;
  e = DecodeTlvElementExpectType(&d1, TT_Name, &nameEle);
  RETURN_IF_ERROR;
  if (unlikely(nameEle.length == 0)) {
    data->name.v = NULL;
    PName_Clear(&data->name.p);
  } else {
    data->name.v = TlvElement_LinearizeValue(&nameEle, pkt, nameMp, &d1);
    RETURN_IF_NULL(data->name.v, NdnError_AllocError);
    e = PName_Parse(&data->name.p, nameEle.length, data->name.v);
    RETURN_IF_ERROR;
  }

  data->freshnessPeriod = 0;
  TlvElement metaEle;
  e = DecodeTlvElementExpectType(&d1, TT_MetaInfo, &metaEle);
  if (e == NdnError_Incomplete || e == NdnError_BadType) {
    return NdnError_OK; // MetaInfo not present
  }
  RETURN_IF_ERROR;

  TlvDecodePos d2;
  TlvElement_MakeValueDecoder(&metaEle, &d2);
  while (!MbufLoc_IsEnd(&d2)) {
    TlvElement metaChild;
    e = DecodeTlvElement(&d2, &metaChild);
    RETURN_IF_ERROR;

    if (metaChild.type != TT_FreshnessPeriod) {
      continue; // ignore other children of MetaInfo
    }

    uint64_t fpV;
    bool ok = TlvElement_ReadNonNegativeInteger(&metaChild, &fpV);
    RETURN_IF_ERROR;
    if (unlikely(fpV > UINT32_MAX)) {
      data->freshnessPeriod = UINT32_MAX;
    } else {
      data->freshnessPeriod = (uint32_t)fpV;
    }
    break;
  }

  return NdnError_OK;
}

void
DataDigest_Prepare(Packet* npkt, struct rte_crypto_op* op)
{
  PData* data = Packet_GetDataHdr(npkt);
  struct rte_mbuf* pkt = Packet_ToMbuf(npkt);
  CryptoOp_PrepareSha256Digest(op, pkt, 0, data->size, data->digest);
}

Packet*
DataDigest_Finish(struct rte_crypto_op* op)
{
  if (unlikely(op->status != RTE_CRYPTO_OP_STATUS_SUCCESS)) {
    rte_pktmbuf_free(op->sym->m_src);
    rte_crypto_op_free(op);
    return NULL;
  }

  Packet* npkt = Packet_FromMbuf(op->sym->m_src);
  PData* data = Packet_GetDataHdr(npkt);
  data->hasDigest = true;
  return npkt;
}
