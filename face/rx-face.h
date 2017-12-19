#ifndef NDN_DPDK_FACE_FACE_H
#define NDN_DPDK_FACE_FACE_H

#include "common.h"

/// \file

/** \brief Network interface for receiving NDN packets.
 */
typedef struct RxFace
{
  uint16_t port;
  uint16_t queue;

  uint64_t nFrames;       ///< number of L2 frames
  uint64_t nInterestPkts; ///< number of Interests decoded
  uint64_t nDataPkts;     ///< number of Data decoded
} RxFace;

static inline uint16_t
RxFace_GetPktPrivSize()
{
  return sizeof(union {
    InterestPkt i;
    DataPkt d;
  });
}

/** \brief Receive and decode a burst of packet.
 *  \param face the face
 *  \param pkts array of packet pointers
 *  \param nPkts size of \p pkt array
 *  \return number of received packets
 *
 *  If a packet has failed decoding, or is retained in reassembly buffer, it is still counted in
 *  the return value, but pkts[i] is set to NULL.
 */
uint16_t RxFace_RxBurst(RxFace* face, struct rte_mbuf** pkts, uint16_t nPkts);

#endif // NDN_DPDK_FACE_FACE_H
