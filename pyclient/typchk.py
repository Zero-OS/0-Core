
missing = object()


def primitive(typ):
    return typ in [str, int, float, bool]


class CheckerException(BaseException):
    pass


class Option:
    def __init__(self):
        raise NotImplementedError()

    def check(self, object):
        raise NotImplementedError()


class Or(Option):
    def __init__(self, *types):
        self._checkers = []
        for typ in types:
            self._checkers.append(Checker(typ))

    def check(self, object):
        for chk in self._checkers:
            if chk.check(object) is True:
                return True
        return False


class IsNone(Option):
    def __init__(self):
        pass

    def check(self, object):
        return object is None


class Missing(Option):
    def __init__(self):
        pass

    def check(self, object):
        return object == missing


class Any(Option):
    def __init__(self):
        pass

    def check(self, object):
        return True


class Length(Option):
    def __init__(self, typ, len):
        self._checker = Checker(typ)
        self._len = len

    def check(self, object):
        if not self._checker.check(object):
            return False
        return len(object) == self._len


class Map(Option):
    def __init__(self, key_type, value_type):
        self._key = Checker(key_type)
        self._value = Checker(value_type)

    def check(self, object):
        if not isinstance(object, dict):
            return False
        for k, v in object.items():
            if not self._key.check(k):
                return False
            if not self._value.check(v):
                return False
        return True


class Checker:
    """
    Build a type checker to check method inputs

    A Checker takes a type definition as following

    c = Checker(<type-def>)
    then use c to check inputs as

    valid = c.check(value)

    type-def:
    - primitive types (str, bool, int, float)
    - composite types ([str], [int], etc...)
    - dicts types ({'name': str, 'age': float, etc...})

    To build a more complex type-def u can use the available Options in typechk module

    - Or(type-def, type-def, ...)
    - Missing() (Only make sense in dict types)
    - IsNone() (accept None value)

    Example of type definition
    A dict object, with the following attributes
    - `name` of type string
    - optional `age` which can be int, or float
    - A list of children each has
        - string name
        - float age


    c = Checker({
        'name': str,
        'age': Or(int, float, Missing()),
        'children': [{'name': str, 'age': float}]
    })

    c.check({'name': 'azmy', 'age': 34, children:[]}) # passes
    c.check({'name': 'azmy', children:[]}) # passes
    c.check({'age': 34, children:[]}) # does not pass
    c.check({'name': 'azmy', children:[{'name': 'yahia', 'age': 4.0}]}) # passes
    c.check({'name': 'azmy', children:[{'name': 'yahia', 'age': 4.0}, {'name': 'yassine'}]}) # does not pass
    """
    def __init__(self, tyepdef):
        self._typ = tyepdef

    def check(self, object):
        return self._check(self._typ, object)

    def _check_list(self, typ, obj_list):
        for elem in obj_list:
            if not self._check(typ, elem):
                return False
        return True

    def _check_dict(self, typ, obj_dict):
        given = []
        for name, value in obj_dict.items():
            if name not in typ:
                return False
            given.append(name)
            attr_type = typ[name]
            if not self._check(attr_type, value):
                return False

        if len(given) == len(typ):
            return True

        type_keys = list(typ.keys())
        for key in given:
            type_keys.remove(key)

        for required in type_keys:
            if not self._check(typ[required], missing):
                return False
        return True

    def _check(self, typ, object):
        if isinstance(typ, Option):
            return typ.check(object)

        atyp = type(object)
        if primitive(atyp):
            return atyp == typ

        if isinstance(typ, (list, tuple)):
            if atyp != list:
                return False
            return self._check_list(typ[0], object)
        elif isinstance(typ, dict):
            if atyp != dict:
                return False
            return self._check_dict(typ, object)

        return False

