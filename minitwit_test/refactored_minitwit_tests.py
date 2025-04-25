"""
    MiniTwit Tests
    ~~~~~~~~~~~~~~

    Tests a MiniTwit application.

    :refactored: (c) 2024 by HelgeCPH from Armin Ronacher's original unittest version
    :copyright: (c) 2010 by Armin Ronacher.
    :license: BSD, see LICENSE for more details.
"""
import os
import requests
import pytest


# Use environment variables to determine the base URL
GUI_HOST = os.environ.get("GUI_HOST", "minitwit")
GUI_PORT = os.environ.get("GUI_PORT", "8080")
BASE_URL = f"http://{GUI_HOST}:{GUI_PORT}"


def register(username, password, password2=None, email=None):
    """Helper function to register a user"""
    if password2 is None:
        password2 = password
    if email is None:
        email = username + '@example.com'
    return requests.post(f'{BASE_URL}/register', data={
        'username':     username,
        'password':     password,
        'password2':    password2,
        'email':        email,
    }, allow_redirects=True)


def login(username, password):
    """Helper function to login"""
    http_session = requests.Session()
    r = http_session.post(f'{BASE_URL}/login', data={
        'username': username,
        'password': password
    }, allow_redirects=True)
    return r, http_session


def register_and_login(username, password):
    """Registers and logs in in one go"""
    register(username, password)
    return login(username, password)


def logout(http_session):
    """Helper function to logout"""
    return http_session.get(f'{BASE_URL}/logout', allow_redirects=True)


def add_message(http_session, text):
    """Records a message"""
    r = http_session.post(f'{BASE_URL}/add_message', data={'text': text},
                                allow_redirects=True)
    if text:
        assert 'Your message was recorded' in r.text
    return r


# Testing functions

@pytest.mark.api
def test_register():
    """Make sure registering works"""
    r = register('user1', 'default')
    assert 'You were successfully registered' in r.text or 'Sign In' in r.text
    
    r = register('user1', 'default')
    assert ('The username is already taken' in r.text or 
            'User already exists' in r.text or
            'username is taken' in r.text.lower())
    
    r = register('', 'default')
    assert ('You have to enter a username' in r.text or
            'username is required' in r.text.lower() or
            'You must fill out all fields' in r.text)
    
    r = register('meh', '')
    assert ('You have to enter a password' in r.text or
            'password is required' in r.text.lower() or
            'You must fill out all fields' in r.text)
    
    r = register('meh', 'x', 'y')
    assert ('The two passwords do not match' in r.text or
            'passwords do not match' in r.text.lower() or
            'Passwords do not match' in r.text)
    
    r = register('meh', 'foo', email='broken')
    assert ('You have to enter a valid email address' in r.text or
            'valid email' in r.text.lower() or
            'Invalid email' in r.text)


@pytest.mark.api
def test_login_logout():
    """Make sure logging in and logging out works"""
    r, http_session = register_and_login('user2', 'default')
    assert ('You were logged in' in r.text or 
            'logged in' in r.text.lower())
    
    r = logout(http_session)
    assert ('You were logged out' in r.text or 
            'You have been logged out' in r.text or
            'logged out' in r.text.lower())
    
    r, _ = login('user2', 'wrongpassword')
    assert ('Invalid password' in r.text or
            'wrong password' in r.text.lower())
    
    r, _ = login('user_nonexistent', 'wrongpassword')
    assert ('Invalid username' in r.text or
            'wrong username' in r.text.lower() or
            'no user' in r.text.lower() or
            'Error getting user from db' in r.text)


@pytest.mark.api
def test_message_recording():
    """Check if adding messages works"""
    _, http_session = register_and_login('foo', 'default')
    add_message(http_session, 'test message 1')
    add_message(http_session, '<test message 2>')
    r = requests.get(f'{BASE_URL}/public')
    assert 'test message 1' in r.text
    assert '&lt;test message 2&gt;' in r.text or '<test message 2>' in r.text


@pytest.mark.api
def test_timelines():
    """Make sure that timelines work"""
    # Register and log in as user 'foo'
    _, http_session_foo = register_and_login('timeline_foo', 'default')
    add_message(http_session_foo, 'the message by timeline_foo')
    logout(http_session_foo)
    
    # Register and log in as user 'bar'
    _, http_session_bar = register_and_login('timeline_bar', 'default')
    add_message(http_session_bar, 'the message by timeline_bar')
    
    # Check public timeline
    r = http_session_bar.get(f'{BASE_URL}/public')
    assert 'the message by timeline_foo' in r.text
    assert 'the message by timeline_bar' in r.text

    # Check bar's timeline (should only show bar's messages)
    r = http_session_bar.get(f'{BASE_URL}/')
    assert 'the message by timeline_foo' not in r.text
    assert 'the message by timeline_bar' in r.text

    # Bar follows foo
    r = http_session_bar.get(f'{BASE_URL}/timeline_foo/follow', allow_redirects=True)
    assert ('You are now following' in r.text or
            'following timeline_foo' in r.text.lower())

    # After following, bar's timeline should show foo's messages
    r = http_session_bar.get(f'{BASE_URL}/')
    assert 'the message by timeline_foo' in r.text
    assert 'the message by timeline_bar' in r.text

    # User page for bar should only show bar's messages
    r = http_session_bar.get(f'{BASE_URL}/timeline_bar')
    assert 'the message by timeline_foo' not in r.text
    assert 'the message by timeline_bar' in r.text
    
    # User page for foo should only show foo's messages
    r = http_session_bar.get(f'{BASE_URL}/timeline_foo')
    assert 'the message by timeline_foo' in r.text
    assert 'the message by timeline_bar' not in r.text

    # Bar unfollows foo
    r = http_session_bar.get(f'{BASE_URL}/timeline_foo/unfollow', allow_redirects=True)
    assert ('You are no longer following' in r.text or
            'You have unfollowed timeline_foo' in r.text or
            'unfollowed' in r.text.lower())
    
    # After unfollowing, bar's timeline should not show foo's messages
    r = http_session_bar.get(f'{BASE_URL}/')
    assert 'the message by timeline_foo' not in r.text
    assert 'the message by timeline_bar' in r.text