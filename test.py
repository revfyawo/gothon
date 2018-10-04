# Comment line
"""Docstring
really long one
""" 'and continuation'
integer = 1
boolean = True
dictionary = {
    'key': 'value'
}

string_literal = 'string literal'
other_string_literal = "other string literal"
format_string = f'format string'
raw_string = r'raw string\t'
raw_format_string = rf'raw format {string_literal}'
bytes_literal = b'bytes literal'
raw_bytes_literal = rb'raw bytes literal'

multiple_part_string = 'first part' r"second part"
multiple_line_string_explicit = 'first line' \
    f'second line'
multiple_line_string_implicit = (
    'first line'
    r'second line'
)

if boolean \
        and integer:
    integer += 1
    boolean = False

print(integer)
