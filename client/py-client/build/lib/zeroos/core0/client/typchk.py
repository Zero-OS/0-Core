
missing = object()


def primitive(typ):
    return typ in [str, int, float, bool]


class CheckerException(Exception):
    pass


class Tracker(Exception):
    def __init__(self, base):
        self._base = base
        self._reason = None
        self._branches = []

    @property
    def branches(self):
        return self._branches

    def copy(self):
        l = self._base.copy()
        t = Tracker(l)
        t._reason = self._reason
        return t

    def push(self, s):
        t = self.copy()
        t._base.append(str(s))
        return t

    def pop(self):
        t = self.copy()
        t._base.pop()
        return t

    def reason(self, reason):
        self._reason = reason
        return self

    def branch(self, tracker):
        t = tracker.copy()
        self._branches.append(t)

    def __str__(self):
        u = "/".join(self._base)
        if self._reason is not None:
            u = '[{}] at -> {}'.format(self._reason, u)
        for branch in self.branches:
            u += '\n  -> {}'.format(branch)

        return u

    def __repr__(self):
        return str(self)


class Option:
    def __init__(self):
        raise NotImplementedError()

    def check(self, object, t):
        raise NotImplementedError()


class Or(Option):
    def __init__(self, *types):
        self._checkers = []
        for typ in types:
            self._checkers.append(Checker(typ))

    def check(self, object, t):
        bt = t.copy()
        for chk in self._checkers:
            try:
                chk.check(object, bt)
                return
            except Tracker as tx:
                t.branch(tx)
        raise t.reason('all branches failed')


class IsNone(Option):
    def __init__(self):
        pass

    def check(self, object, t):
        if object is not None:
            raise t.reason('is not none')


class Missing(Option):
    def __init__(self):
        pass

    def check(self, object, t):
        if object != missing:
            raise t.reason('is not missing')


class Any(Option):
    def __init__(self):
        pass

    def check(self, object, t):
        return


class Length(Option):
    def __init__(self, typ, min=None, max=None):
        self._checker = Checker(typ)
        if min is None and max is None:
            raise ValueError("you have to pass wither min or max to the length type checker")
        self._min = min
        self._max = max

    def check(self, object, t):
        self._checker.check(object, t)
        if self._min is not None and len(object) < self._min:
            raise t.reason('invalid length, expecting more than or equal {} got {}'.format(self._min, len(object)))
        if self._max is not None and len(object) > self._max:
            raise t.reason('invalid length, expecting less than or equal {} got {}'.format(self._max, len(object)))


class Map(Option):
    def __init__(self, key_type, value_type):
        self._key = Checker(key_type)
        self._value = Checker(value_type)

    def check(self, object, t):
        if not isinstance(object, dict):
            raise t.reason('expecting a dict, got {}'.format(type(object)))
        for k, v in object.items():
            tx = t.push(k)
            self._key.check(k, tx)
            tv = t.push('{}[value]'.format(k))
            self._value.check(v, tv)


class Enum(Option):
    def __init__(self, *valid):
        self._valid = valid

    def check(self, object, t):
        if not isinstance(object, str):
            raise t.reason('expecting string, got {}'.format(type(object)))
        if object not in self._valid:
            raise t.reason('value "{}" not in enum'.format(object))


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

    def check(self, object, tracker=None):
        if tracker is None:
            tracker = Tracker([]).push('/')
        return self._check(self._typ, object, tracker)

    def _check_list(self, typ, obj_list, t):
        for i, elem in enumerate(obj_list):
            tx = t.push('[{}]'.format(i))
            self._check(typ, elem, tx)

    def _check_dict(self, typ, obj_dict, t):
        given = []
        for name, value in obj_dict.items():
            tx = t.push(name)
            if name not in typ:
                raise tx.reason('unknown key "{}"'.format(name))
            given.append(name)
            attr_type = typ[name]
            self._check(attr_type, value, tx)

        if len(given) == len(typ):
            return

        type_keys = list(typ.keys())
        for key in given:
            type_keys.remove(key)

        for required in type_keys:
            tx = t.push(required)
            self._check(typ[required], missing, tx)

    def _check(self, typ, object, t):
        if isinstance(typ, Option):
            return typ.check(object, t)

        atyp = type(object)
        if isinstance(typ, list):
            if atyp != list:
                raise t.reason('expecting a list')
            self._check_list(typ[0], object, t)
        elif isinstance(typ, tuple):
            if atyp != tuple:
                raise t.reason('expecting a tuple')
            self._check_list(typ[0], object, t)
        elif isinstance(typ, dict):
            if atyp != dict:
                raise t.reason('expecting a dict')
            self._check_dict(typ, object, t)
        elif atyp != typ:
            raise t.reason('invalid type, expecting {}'.format(typ))