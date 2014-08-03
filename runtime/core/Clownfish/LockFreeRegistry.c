/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#define C_CFISH_LOCKFREEREGISTRY
#define CFISH_USE_SHORT_NAMES

#include "Clownfish/LockFreeRegistry.h"
#include "Clownfish/Err.h"
#include "Clownfish/Class.h"
#include "Clownfish/Util/Atomic.h"
#include "Clownfish/Util/Memory.h"

typedef struct cfish_LFRegEntry {
    Obj *key;
    Obj *value;
    int32_t hash_sum;
    struct cfish_LFRegEntry *volatile next;
} cfish_LFRegEntry;
#define LFRegEntry cfish_LFRegEntry

LockFreeRegistry*
LFReg_new(size_t capacity) {
    LockFreeRegistry *self
        = (LockFreeRegistry*)Class_Make_Obj(LOCKFREEREGISTRY);
    return LFReg_init(self, capacity);
}

LockFreeRegistry*
LFReg_init(LockFreeRegistry *self, size_t capacity) {
    self->capacity = capacity;
    self->entries  = CALLOCATE(capacity, sizeof(void*));
    return self;
}

bool
LFReg_Register_IMP(LockFreeRegistry *self, Obj *key, Obj *value) {
    LFRegEntry  *new_entry = NULL;
    int32_t      hash_sum  = Obj_Hash_Sum(key);
    size_t       bucket    = (uint32_t)hash_sum  % self->capacity;
    LFRegEntry  *volatile *entries = (LFRegEntry*volatile*)self->entries;
    LFRegEntry  *volatile *slot    = &(entries[bucket]);

    // Proceed through the linked list.  Bail out if the key has already been
    // registered.
FIND_END_OF_LINKED_LIST:
    while (*slot) {
        LFRegEntry *entry = *slot;
        if (entry->hash_sum == hash_sum) {
            if (Obj_Equals(key, entry->key)) {
                return false;
            }
        }
        slot = &(entry->next);
    }

    // We've found an empty slot. Create the new entry.
    if (!new_entry) {
        new_entry = (LFRegEntry*)MALLOCATE(sizeof(LFRegEntry));
        new_entry->hash_sum  = hash_sum;
        new_entry->key       = INCREF(key);
        new_entry->value     = INCREF(value);
        new_entry->next      = NULL;
    }

    /* Attempt to append the new node onto the end of the linked list.
     * However, if another thread filled the slot since we found it (perhaps
     * while we were allocating that new node), the compare-and-swap will
     * fail.  If that happens, we have to go back and find the new end of the
     * linked list, then try again. */
    if (!Atomic_cas_ptr((void*volatile*)slot, NULL, new_entry)) {
        goto FIND_END_OF_LINKED_LIST;
    }

    return true;
}

Obj*
LFReg_Fetch_IMP(LockFreeRegistry *self, Obj *key) {
    int32_t      hash_sum  = Obj_Hash_Sum(key);
    size_t       bucket    = (uint32_t)hash_sum  % self->capacity;
    LFRegEntry **entries   = (LFRegEntry**)self->entries;
    LFRegEntry  *entry     = entries[bucket];

    while (entry) {
        if (entry->hash_sum  == hash_sum) {
            if (Obj_Equals(key, entry->key)) {
                return entry->value;
            }
        }
        entry = entry->next;
    }

    return NULL;
}

LockFreeRegistry*
LFReg_Clone_IMP(LockFreeRegistry *self) {
    size_t             capacity = self->capacity;
    LFRegEntry       **entries  = (LFRegEntry**)self->entries;
    LockFreeRegistry  *twin     = LFReg_new(capacity);

    for (size_t i = 0; i < capacity; ++i) {
        LFRegEntry *entry = entries[i];

        while (entry) {
            Obj *key   = Obj_Clone(entry->key);
            Obj *value = Obj_Clone(entry->value);
            LFReg_Register(twin, key, value);
            DECREF(key);
            DECREF(value);

            entry = entry->next;
        }
    }

    return twin;
}

void
LFReg_Destroy_IMP(LockFreeRegistry *self) {
    LFRegEntry **entries = (LFRegEntry**)self->entries;

    for (size_t i = 0; i < self->capacity; i++) {
        LFRegEntry *entry = entries[i];
        while (entry) {
            LFRegEntry *next_entry = entry->next;
            DECREF(entry->key);
            DECREF(entry->value);
            FREEMEM(entry);
            entry = next_entry;
        }
    }
    FREEMEM(self->entries);

    SUPER_DESTROY(self, LOCKFREEREGISTRY);
}


