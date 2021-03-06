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

#ifndef H_CFCRUBY
#define H_CFCRUBY

#ifdef __cplusplus
extern "C" {
#endif

typedef struct CFCRuby CFCRuby;
struct CFCParcel;
struct CFCHierarchy;

/** Clownfish::CFC::Binding::Ruby - Perl bindings for a
 * Clownfish::CFC::Model::Hierarchy.
 * 
 * Clownfish::CFC::Binding::Ruby presents an interface for auto-generating XS
 * and Perl code to bind C code for a Clownfish class hierarchy to Perl.
 * 
 * In theory this module could be much more flexible and its API could be more
 * elegant.  There are many ways which you could walk the parsed parcels,
 * classes, methods, etc. in a Clownfish::CFC::Model::Hierarchy and generate
 * binding code.  However, our needs are very limited, so we are content with
 * a "one size fits one" solution.
 * 
 * In particular, this module assumes that the XS bindings for all classes in
 * the hierarchy should be assembled into a single shared object which belongs
 * to the primary, "boot" class.  There's no reason why it could not write one
 * .xs file per class, or one per parcel, instead.
 * 
 * The files written by this class are derived from the name of the boot class.
 * If it is "Crustacean", the following files will be generated.
 * 
 *     # Generated by write_bindings()
 *     $lib_dir/Crustacean.xs
 * 
 *     # Generated by write_boot()
 *     $hierarchy_dest_dir/crust_boot.h
 *     $hierarchy_dest_dir/crust_boot.c
 */

/** 
 * @param parcel The L<Clownfish::CFC::Model::Parcel> to which the
 * C<boot_class> belongs.
 * @param hierarchy A Clownfish::CFC::Model::Hierarchy.
 * @param lib_dir location of the Perl lib directory to which files will be
 * written.
 * @param boot_class The name of the main class, which will own the shared
 * object.
 * @param header Text which will be prepended to generated C/XS files --
 * typically, an "autogenerated file" warning.
 * @param footer Text to be appended to the end of generated C/XS files --
 * typically copyright information.
 */
CFCRuby*
CFCRuby_new(struct CFCParcel *parcel, struct CFCHierarchy *hierarchy,
            const char *lib_dir, const char *boot_class, const char *header,
            const char *footer);

CFCRuby*
CFCRuby_init(CFCRuby *self, struct CFCParcel *parcel,
             struct CFCHierarchy *hierarchy, const char *lib_dir,
             const char *boot_class, const char *header, const char *footer);

void
CFCRuby_destroy(CFCRuby *self);

/** Write out "boot" files to the Hierarchy's "dest_dir" which contain code
 * for bootstrapping Clownfish classes.
 */
void
CFCRuby_write_boot(CFCRuby *self);

/** Write out cfish_hostdefs.h file.
 */
void
CFCRuby_write_hostdefs(CFCRuby *self);

#ifdef __cplusplus
}
#endif

#endif /* H_CFCPERL */

