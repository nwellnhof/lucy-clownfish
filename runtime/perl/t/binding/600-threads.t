# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

use strict;
use warnings;

use threads;

use Clownfish;
use Test::More tests => 7;

sub cf_addr {
    my $obj = shift;
    return 0 + $$obj;
}

my $obj           = Clownfish::String->new('A string.');
my $registry_addr = cf_addr(Clownfish::Class->_get_registry);
my $class         = Clownfish::Class->fetch_class('Clownfish::Hash');
my $class_addr    = cf_addr($class);
my $parent        = $class->get_parent;
my $parent_addr   = cf_addr($parent);

my ($thread) = threads->create(sub {
    my $thr_registry = Clownfish::Class->_get_registry;
    my $thr_class    = Clownfish::Class->fetch_class('Clownfish::Hash');
    my $thr_parent   = $thr_class->get_parent;
    my $thr_other_parent
        = Clownfish::Class->fetch_class($thr_parent->get_name);
    return (
        defined($$obj),
        cf_addr($thr_registry),
        cf_addr($thr_class),
        cf_addr($thr_parent),
        cf_addr($thr_other_parent),
    );
});
my (
    $thr_obj_defined,
    $thr_registry_addr,
    $thr_class_addr,
    $thr_parent_addr,
    $thr_other_parent_addr,
) = $thread->join;

ok( !$thr_obj_defined, "Object is undefined in other thread" );

my $other_registry_addr = cf_addr(Clownfish::Class->_get_registry);
my $other_class         = Clownfish::Class->fetch_class('Clownfish::Hash');
my $other_class_addr    = cf_addr($class);

is( $other_registry_addr, $registry_addr, "Same registry in same thread" );
is( $other_class_addr, $class_addr, "Same class in same thread" );
isnt( $thr_registry_addr, $registry_addr, "Cloned registry in other thread" );
isnt( $thr_class_addr, $class_addr, "Cloned class in other thread" );
isnt( $thr_parent_addr, $parent_addr, "Cloned parent class in other thread" );
is( $thr_parent_addr, $thr_other_parent_addr, "Parent classes fixed up" );

