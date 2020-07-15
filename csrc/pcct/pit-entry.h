#ifndef NDNDPDK_PCCT_PIT_ENTRY_H
#define NDNDPDK_PCCT_PIT_ENTRY_H

/** @file */

#include "../fib/fib.h"
#include "pit-dn.h"
#include "pit-struct.h"
#include "pit-up.h"

#define PIT_ENTRY_MAX_DNS 6
#define PIT_ENTRY_MAX_UPS 2
#define PIT_ENTRY_EXT_MAX_DNS 16
#define PIT_ENTRY_EXT_MAX_UPS 8
#define PIT_ENTRY_SG_SCRATCH 64

#define PIT_ENTRY_FIBPREFIXL_NBITS_ 9
static_assert((1 << PIT_ENTRY_FIBPREFIXL_NBITS_) > FibMaxNameLength, "");

typedef struct PitEntryExt PitEntryExt;

/**
 * @brief A PIT entry.
 *
 * This struct is enclosed in @c PccEntry .
 */
struct PitEntry
{
  Packet* npkt;   ///< representative Interest packet
  MinTmr timeout; ///< timeout timer
  TscTime expiry; ///< when all DNs expire

  uint64_t fibPrefixHash;                            ///< hash value of FIB prefix
  uint32_t fibSeqNum;                                ///< FIB entry sequence number
  uint8_t nCanBePrefix;                              ///< how many DNs want CanBePrefix?
  uint8_t txHopLimit;                                ///< HopLimit for outgoing Interests
  uint16_t fibPrefixL : PIT_ENTRY_FIBPREFIXL_NBITS_; ///< TLV-LENGTH of FIB prefix
  bool mustBeFresh : 1;                              ///< entry for MustBeFresh 0 or 1?
  bool hasSgTimer : 1; ///< whether timeout is set by strategy or expiry

  PitEntryExt* ext;
  PitDn dns[PIT_ENTRY_MAX_DNS];
  PitUp ups[PIT_ENTRY_MAX_UPS];

  char sgScratch[PIT_ENTRY_SG_SCRATCH];
};
static_assert(offsetof(PitEntry, dns) <= RTE_CACHE_LINE_SIZE, "");

struct PitEntryExt
{
  PitDn dns[PIT_ENTRY_EXT_MAX_DNS];
  PitUp ups[PIT_ENTRY_EXT_MAX_UPS];
  PitEntryExt* next;
};

__attribute__((nonnull)) static inline void
PitEntry_SetFibEntry_(PitEntry* entry, PInterest* interest, const FibEntry* fibEntry)
{
  entry->fibPrefixL = fibEntry->nameL;
  entry->fibSeqNum = fibEntry->seqNum;
  PName* name = &interest->name;
  if (unlikely(interest->activeFwHint >= 0)) {
    name = &interest->fwHint;
  }
  entry->fibPrefixHash = PName_ComputePrefixHash(name, fibEntry->nComps);
  memset(entry->sgScratch, 0, PIT_ENTRY_SG_SCRATCH);
}

/**
 * @brief Initialize a PIT entry.
 * @param npkt the Interest packet.
 */
__attribute__((nonnull)) static inline void
PitEntry_Init(PitEntry* entry, Packet* npkt, const FibEntry* fibEntry)
{
  PInterest* interest = Packet_GetInterestHdr(npkt);
  entry->npkt = npkt;
  MinTmr_Init(&entry->timeout);
  entry->expiry = 0;

  entry->nCanBePrefix = interest->canBePrefix;
  entry->txHopLimit = 0;
  entry->mustBeFresh = interest->mustBeFresh;

  entry->dns[0].face = 0;
  entry->ups[0].face = 0;
  entry->ext = NULL;

  PitEntry_SetFibEntry_(entry, interest, fibEntry);
}

/** @brief Finalize a PIT entry. */
__attribute__((nonnull)) static inline void
PitEntry_Finalize(PitEntry* entry)
{
  if (likely(entry->npkt != NULL)) {
    rte_pktmbuf_free(Packet_ToMbuf(entry->npkt));
  }
  MinTmr_Cancel(&entry->timeout);
  for (PitEntryExt* ext = entry->ext; unlikely(ext != NULL);) {
    PitEntryExt* next = ext->next;
    rte_mempool_put(rte_mempool_from_obj(ext), ext);
    ext = next;
  }
}

/**
 * @brief Represent PIT entry as a string for debug purpose.
 * @return A string from thread-local buffer.
 * @warning Subsequent *ToDebugString calls on the same thread overwrite the buffer.
 */
__attribute__((nonnull, returns_nonnull)) const char*
PitEntry_ToDebugString(PitEntry* entry);

/** @brief Get a token that identifies the PIT entry. */
__attribute__((nonnull)) static inline uint64_t
PitEntry_GetToken(PitEntry* entry);
// Implementation is in pit.h to avoid circular dependency.

/**
 * @brief Reference FIB entry from PIT entry, clear scratch if FIB entry changed.
 * @param npkt the Interest packet.
 */
__attribute__((nonnull)) static inline void
PitEntry_RefreshFibEntry(PitEntry* entry, Packet* npkt, const FibEntry* fibEntry)
{
  if (likely(entry->fibSeqNum == fibEntry->seqNum)) {
    return;
  }

  PInterest* interest = Packet_GetInterestHdr(npkt);
  PitEntry_SetFibEntry_(entry, interest, fibEntry);
}

/**
 * @brief Retrieve FIB entry via PIT entry's FIB reference.
 * @pre Calling thread holds rcu_read_lock, which must be retained until it stops
 *      using the returned entry.
 */
__attribute__((nonnull)) FibEntry*
PitEntry_FindFibEntry(PitEntry* entry, Fib* fib);

/** @brief Set timer to erase PIT entry when its last PitDn expires. */
__attribute__((nonnull)) void
PitEntry_SetExpiryTimer(PitEntry* entry, Pit* pit);

/**
 * @brief Set timer to invoke strategy after @p after.
 * @retval Timer set successfully.
 * @retval Unable to set timer; reverted to expiry timer.
 */
__attribute__((nonnull)) bool
PitEntry_SetSgTimer(PitEntry* entry, Pit* pit, TscDuration after);

__attribute__((nonnull)) void
PitEntry_Timeout_(MinTmr* tmr, void* pit0);

/**
 * @brief Find duplicate nonce among DN records other than @p rxFace.
 * @return FaceID of PitDn with duplicate nonce, or zero if none.
 */
__attribute__((nonnull)) FaceID
PitEntry_FindDuplicateNonce(PitEntry* entry, uint32_t nonce, FaceID rxFace);

/**
 * @brief Insert new DN record, or update existing DN record.
 * @param entry PIT entry, must be initialized.
 * @param npkt received Interest; will take ownership unless returning NULL.
 * @return DN record, or NULL if no slot is available.
 */
__attribute__((nonnull)) PitDn*
PitEntry_InsertDn(PitEntry* entry, Pit* pit, Packet* npkt);

/**
 * @brief Find existing UP record, or reserve slot for new UP record.
 * @param entry PIT entry, must be initialized.
 * @param face upstream face.
 * @return UP record, or NULL if no slot is available.
 * @note If returned UP record is unused (no @c PitUp_RecordTx invocation),
 *       it will be overwritten on the next @c PitEntry_ReserveUp invocation.
 */
__attribute__((nonnull)) PitUp*
PitEntry_ReserveUp(PitEntry* entry, Pit* pit, FaceID face);

/**
 * @brief Calculate InterestLifetime for TX Interest.
 * @return InterestLifetime in millis.
 */
__attribute__((nonnull)) static inline uint32_t
PitEntry_GetTxInterestLifetime(PitEntry* entry, TscTime now)
{
  return TscDuration_ToMillis(entry->expiry - now);
}

/** @brief Calculate HopLimit for TX Interest. */
__attribute__((nonnull)) static inline uint8_t
PitEntry_GetTxInterestHopLimit(PitEntry* entry)
{
  return entry->txHopLimit;
}

#endif // NDNDPDK_PCCT_PIT_ENTRY_H
