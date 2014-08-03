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

package Clownfish;

use 5.008003;

our $VERSION = '0.003000';
$VERSION = eval $VERSION;

use Exporter 'import';
BEGIN {
    our @EXPORT_OK = qw(
        to_clownfish
        to_perl
        kdump
        );
}

# On most UNIX variants, this flag makes DynaLoader pass RTLD_GLOBAL to
# dl_open, so extensions can resolve the needed symbols without explicitly
# linking against the DSO.
sub dl_load_flags { 1 }

BEGIN {
    require DynaLoader;
    our @ISA = qw( DynaLoader );
    # This loads a large number of disparate subs.
    bootstrap Clownfish '0.3.0';
}

sub kdump {
    require Data::Dumper;
    my $kdumper = Data::Dumper->new( [@_] );
    $kdumper->Sortkeys( sub { return [ sort keys %{ $_[0] } ] } );
    $kdumper->Indent(1);
    warn $kdumper->Dump;
}

sub error {$Clownfish::Err::error}

{
    package Clownfish::Util::StringHelper;
    our $VERSION = '0.003000';
    $VERSION = eval $VERSION;
    BEGIN {
        push our @ISA, 'Exporter';
        our @EXPORT_OK = qw(
            utf8_flag_on
            utf8_flag_off
            to_base36
            from_base36
            utf8ify
            utf8_valid
            cat_bytes
        );
    }
}

{
    package Clownfish::LockFreeRegistry;
    our $VERSION = '0.003000';
    $VERSION = eval $VERSION;
    no warnings 'redefine';
    # Clone LFReg manually.
    sub CLONE_SKIP { 0; }
    sub DESTROY { }    # leak all
}

{
    package Clownfish::Obj;
    our $VERSION = '0.003000';
    $VERSION = eval $VERSION;
    use Clownfish qw( to_clownfish to_perl );
    use Carp qw( confess );
    # Clownfish objects are not thread-safe.
    sub CLONE_SKIP { 1; }
    sub STORABLE_freeze {
        my $class_name = ref(shift);
        confess("Storable serialization not implemented for $class_name");
    }
    sub STORABLE_thaw {
        my $class_name = ref(shift);
        confess("Storable serialization not implemented for $class_name");
    }
}

{
    package Clownfish::Class;
    our $VERSION = '0.003000';
    $VERSION = eval $VERSION;

    sub _find_parent_class {
        my $package = shift;
        no strict 'refs';
        for my $parent ( @{"$package\::ISA"} ) {
            return $parent if $parent->isa('Clownfish::Obj');
        }
        return;
    }

    sub _fresh_host_methods {
        my $package = shift;
        no strict 'refs';
        my $stash = \%{"$package\::"};
        my $methods
            = Clownfish::VArray->new( capacity => scalar keys %$stash );
        while ( my ( $symbol, $glob ) = each %$stash ) {
            next if ref $glob;
            next unless *$glob{CODE};
            $methods->push( Clownfish::String->new($symbol) );
        }
        return $methods;
    }

    sub _register {
        my ( $singleton, $parent ) = @_;
        my $singleton_class = $singleton->get_name;
        my $parent_class    = $parent->get_name;
        if ( !$singleton_class->isa($parent_class) ) {
            no strict 'refs';
            push @{"$singleton_class\::ISA"}, $parent_class;
        }
    }

    no warnings 'redefine';
    sub DESTROY { }    # leak all
}

{
    package Clownfish::Method;
    our $VERSION = '0.003000';
    $VERSION = eval $VERSION;
    no warnings 'redefine';
    sub DESTROY { }    # leak all
}

{
    package Clownfish::ViewByteBuf;
    our $VERSION = '0.003000';
    $VERSION = eval $VERSION;
    use Carp;
    sub new { confess "ViewByteBuf objects can only be created from C." }
}

{
    package Clownfish::String;
    our $VERSION = '0.003000';
    $VERSION = eval $VERSION;

    {
        # Defeat obscure bugs in the XS auto-generation by redefining clone().
        # (Because of how the typemap works for String*,
        # the auto-generated methods return UTF-8 Perl scalars rather than
        # actual String objects.)
        no warnings 'redefine';
        sub clone { shift->_clone(@_) }
    }
}

{
    package Clownfish::StackString;
    our $VERSION = '0.003000';
    $VERSION = eval $VERSION;
    use Carp;
    sub new { confess "StackString objects can only be created from C." }
    no warnings 'redefine';
    sub DESTROY { }
}

{
    package Clownfish::Err;
    our $VERSION = '0.003000';
    $VERSION = eval $VERSION;
    sub do_to_string { shift->to_string }
    use Scalar::Util qw( blessed );
    use Carp qw( confess longmess );
    use overload
        '""'     => \&do_to_string,
        fallback => 1;

    sub new {
        my ( $either, $message ) = @_;
        my ( undef, $file, $line ) = caller;
        $message .= ", $file line $line\n";
        return $either->_new( mess => Clownfish::String->new($message) );
    }

    sub do_throw {
        my $err      = shift;
        my $longmess = longmess();
        $longmess =~ s/^\s*/\t/;
        $err->cat_mess($longmess);
        die $err;
    }

    our $error;
    sub set_error {
        my $val = $_[1];
        if ( defined $val ) {
            confess("Not a Clownfish::Err")
                unless ( blessed($val)
                && $val->isa("Clownfish::Err") );
        }
        $error = $val;
    }
    sub get_error {$error}
}

{
    package Clownfish::VArray;
    our $VERSION = '0.003000';
    $VERSION = eval $VERSION;
    no warnings 'redefine';
    sub clone       { CORE::shift->_clone }
}

1;

__END__

=head1 NAME

Clownfish - Apache Clownfish symbiotic object system.

=head1 VERSION

0.3.0

=head1 DESCRIPTION

The Apache Clownfish "symbiotic" object system for C is designed to pair with
a "host" dynamic language environment, facilitating the development of high
performance host language extensions.  Clownfish classes are declared in
header files with a C<.cfh> extension.  The Clownfish headers are used by the
Clownfish compiler to generate C header files and host language bindings.
Methods, functions and variables are defined in normal C source files.

The API for this alpha release of Clownfish has been cloaked.

=head1 COPYRIGHT

Clownfish is distributed under the Apache License, Version 2.0, as
described in the file C<LICENSE> included with the distribution.

=cut

