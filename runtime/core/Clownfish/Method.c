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

#define C_CFISH_METHOD
#define CFISH_USE_SHORT_NAMES

#include "Clownfish/Method.h"
#include "Clownfish/String.h"
#include "Clownfish/Err.h"
#include "Clownfish/Class.h"

Method*
Method_new(String *name, cfish_method_t callback_func, size_t offset) {
    Method *self = (Method*)Class_Make_Obj(METHOD);
    return Method_init(self, name, callback_func, offset);
}

Method*
Method_init(Method *self, String *name, cfish_method_t callback_func,
            size_t offset) {
    self->name          = (String*)INCREF(name);
    self->host_alias    = NULL;
    self->callback_func = callback_func;
    self->offset        = offset;
    self->is_excluded   = false;
    return self;
}

Method*
Method_Clone_IMP(Method *self) {
    String *name  = Str_Clone(self->name);
    Method *twin = Method_new(name, self->callback_func, self->offset);
    DECREF(name);

    if (self->host_alias) {
        twin->host_alias = Str_Clone(self->host_alias);
    }

    return twin;
}

void
Method_Destroy_IMP(Method *self) {
    THROW(ERR, "Insane attempt to destroy Method '%o'", self->name);
}

Obj*
Method_Inc_RefCount_IMP(Method *self) {
    return (Obj*)self;
}

uint32_t
Method_Dec_RefCount_IMP(Method *self) {
    UNUSED_VAR(self);
    return 1;
}

uint32_t
Method_Get_RefCount_IMP(Method *self) {
    UNUSED_VAR(self);
    // See comments in Class.c
    return 1;
}

String*
Method_Get_Name_IMP(Method *self) {
    return self->name;
}

String*
Method_Get_Host_Alias_IMP(Method *self) {
    return self->host_alias;
}

bool
Method_Is_Excluded_From_Host_IMP(Method *self) {
    return self->is_excluded;
}


